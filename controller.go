package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	clientnative "github.com/haproxytech/client-native"
	"github.com/haproxytech/client-native/configuration"
	"github.com/haproxytech/client-native/runtime"
	"github.com/haproxytech/config-parser"
	"github.com/haproxytech/models"
)

// HAProxyController is ingress controller
type HAProxyController struct {
	k8s               *K8s
	cfg               Configuration
	osArgs            OSArgs
	NativeAPI         *clientnative.HAProxyClient
	NativeParser      parser.Parser
	ActiveTransaction string
}

// Start initialize and run HAProxyController
func (c *HAProxyController) Start(osArgs OSArgs) {

	c.osArgs = osArgs

	c.HAProxyInitialize()

	var k8s *K8s
	var err error
	if osArgs.OutOfCluster {
		k8s, err = GetRemoteKubernetesClient(osArgs)
	} else {
		k8s, err = GetKubernetesClient()
	}
	if err != nil {
		log.Panic(err)
	}
	c.k8s = k8s

	x := k8s.API.Discovery()
	k8sVersion, _ := x.ServerVersion()
	log.Printf("Running on Kubernetes version: %s %s", k8sVersion.String(), k8sVersion.Platform)

	go c.monitorChanges()
}

//HAProxyInitialize runs HAProxy for the first time so native client can have access to it
func (c *HAProxyController) HAProxyInitialize() {
	//cmd := exec.Command("haproxy", "-f", HAProxyCFG)
	err := os.MkdirAll(HAProxyCertDir, 0644)
	if err != nil {
		log.Panic(err.Error())
	}
	err = os.MkdirAll(HAProxyStateDir, 0644)
	if err != nil {
		log.Panic(err.Error())
	}

	log.Println("Starting HAProxy with", HAProxyCFG)
	cmd := exec.Command("service", "haproxy", "start")
	err = cmd.Run()
	if err != nil {
		log.Println(err)
	}

	c.NativeParser = parser.Parser{}
	err = c.NativeParser.LoadData(HAProxyGlobalCFG)
	if err != nil {
		log.Panic(err)
	}

	runtimeClient := runtime.Client{}
	err = runtimeClient.Init([]string{"/var/run/haproxy-runtime-api.sock"})
	if err != nil {
		log.Panicln(err)
	}

	confClient := configuration.Client{}
	err = confClient.Init(configuration.ClientParams{
		ConfigurationFile: HAProxyCFG,
		//GlobalConfigurationFile: HAProxyGlobalCFG,
		Haproxy: "haproxy",
		//LBCTLPath:               "/usr/sbin/lbctl",
	})
	if err != nil {
		log.Panicln(err)
	}

	c.NativeAPI = &clientnative.HAProxyClient{
		Configuration: &confClient,
		Runtime:       &runtimeClient,
	}
}

func (c *HAProxyController) saveServerState() error {
	result, err := c.NativeAPI.Runtime.ExecuteRaw("show servers state")
	if err != nil {
		return err
	}
	var f *os.File
	if f, err = os.Create(HAProxyStateDir + "global"); err != nil {
		log.Println(err)
		return err
	}
	defer f.Close()
	if _, err = f.Write([]byte(result[0])); err != nil {
		log.Println(err)
		return err
	}
	if err = f.Sync(); err != nil {
		log.Println(err)
		return err
	}
	if err = f.Close(); err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func (c *HAProxyController) HAProxyReload() error {
	err := c.NativeParser.Save(HAProxyGlobalCFG)
	if err != nil {
		return err
	}
	err = c.saveServerState()
	LogErr(err)
	//cmd := exec.Command("haproxy", "-f", HAProxyCFG)
	cmd := exec.Command("service", "haproxy", "reload")
	err = cmd.Run()
	return err
}

func (c *HAProxyController) handlePath(index int, namespace *Namespace, ingress *Ingress, rule *IngressRule, path *IngressPath,
	transaction *models.Transaction, backendsUsed map[string]int) error {
	nativeAPI := c.NativeAPI
	//log.Println("PATH", path)
	backendName, selector, service, err := c.handleService(index, namespace, ingress, rule, path, backendsUsed, transaction)
	if err != nil {
		return err
	}
	if numberOfTimesBackendUsed := backendsUsed[backendName]; numberOfTimesBackendUsed > 1 {
		return nil
	}
	annMaxconn, _ := GetValueFromAnnotations("pod-maxconn", service.Annotations)
	annCheck, _ := GetValueFromAnnotations("check", service.Annotations, c.cfg.ConfigMap.Annotations)

	for _, pod := range namespace.Pods {
		if hasSelectors(selector, pod.Labels) {
			if pod.Backends == nil {
				pod.Backends = map[string]struct{}{}
			}
			pod.Backends[backendName] = struct{}{}
			port := int64(path.ServicePort)
			weight := int64(128)
			data := &models.Server{
				Name:    pod.HAProxyName,
				Address: pod.IP,
				Port:    &port,
				Weight:  &weight,
			}
			if pod.Maintenance {
				data.Maintenance = "enabled"
			}
			/*if pod.Sorry != "" {
				data.Sorry = pod.Sorry
			}*/
			annnotationsActive := false
			if annMaxconn != nil {
				if annMaxconn.Status != DELETED {
					if maxconn, err := strconv.ParseInt(annMaxconn.Value, 10, 64); err == nil {
						data.Maxconn = &maxconn
					}
				}
				if annMaxconn.Status != "" {
					annnotationsActive = true
				}
			}
			if annCheck != nil {
				if annCheck.Status != DELETED {
					if annCheck.Value == "enabled" {
						data.Check = "enabled"
						//see if we have port and interval defined
					}
				}
				if annCheck.Status != "" {
					annnotationsActive = true
				}
			}
			if pod.Status == EMPTY && annnotationsActive {
				pod.Status = MODIFIED
			}
			switch pod.Status {
			case ADDED:
				if err := nativeAPI.Configuration.CreateServer(backendName, data, transaction.ID, 0); err != nil {
					return err
				}
			case MODIFIED:
				if err := nativeAPI.Configuration.EditServer(data.Name, backendName, data, transaction.ID, 0); err != nil {
					return err
				}
			case DELETED:
				if err := nativeAPI.Configuration.DeleteServer(data.Name, backendName, transaction.ID, 0); err != nil {
					return err
				}
			}
		} //if pod.Status...
	} //for pod
	return nil
}

func (c *HAProxyController) handleService(index int, namespace *Namespace, ingress *Ingress, rule *IngressRule, path *IngressPath,
	backendsUsed map[string]int, transaction *models.Transaction) (backendName string, selector MapStringW, service *Service, err error) {
	nativeAPI := c.NativeAPI

	service, ok := namespace.Services[path.ServiceName]
	if !ok {
		log.Println("service", path.ServiceName, "does not exists")
		return "", nil, nil, fmt.Errorf("service %s does not exists", path.ServiceName)
	}
	selector = service.Selector
	if len(selector) == 0 {
		return "", nil, nil, fmt.Errorf("service %s has no selector", service.Name)
	}

	backendName = fmt.Sprintf("%s-%s-%d", namespace.Name, service.Name, path.ServicePort)
	backendsUsed[backendName]++
	//load-balance, forwarded-for and annWhitelist have default values, so no need for error checking
	annBalanceAlg, _ := GetValueFromAnnotations("load-balance", service.Annotations, ingress.Annotations, c.cfg.ConfigMap.Annotations)
	annForwardedFor, _ := GetValueFromAnnotations("forwarded-for", service.Annotations, ingress.Annotations, c.cfg.ConfigMap.Annotations)
	annWhitelist, _ := GetValueFromAnnotations("whitelist", service.Annotations, ingress.Annotations, c.cfg.ConfigMap.Annotations)
	annWhitelistRL, _ := GetValueFromAnnotations("whitelist-with-rate-limit", service.Annotations, ingress.Annotations, c.cfg.ConfigMap.Annotations)
	allowRateLimiting := annWhitelistRL.Value != "" && annWhitelistRL.Value != "OFF"
	status := annWhitelist.Status
	if status == "" {
		if annWhitelistRL.Status != EMPTY {
			data, ok := c.cfg.HTTPRequests[fmt.Sprintf("WHT-%d", index)]
			if ok && len(data) > 0 {
				status = MODIFIED
			}
		}
	}
	if status == "" {
		if annWhitelistRL.Value != "" && path.Status == ADDED {
			status = MODIFIED
		}
	}
	switch status {
	case ADDED, MODIFIED:
		if annWhitelist.Value != "" {
			ID := int64(0)
			httpRequest1 := &models.HTTPRequestRule{
				ID:       &ID,
				Type:     "allow",
				Cond:     "if",
				CondTest: fmt.Sprintf("{ path_beg %s } { src %s }", path.Path, strings.Replace(annWhitelist.Value, ",", " ", -1)),
			}
			httpRequest2 := &models.HTTPRequestRule{
				ID:       &ID,
				Type:     "deny",
				Cond:     "if",
				CondTest: fmt.Sprintf("{ path_beg %s }", path.Path),
			}
			if allowRateLimiting {
				c.cfg.HTTPRequests[fmt.Sprintf("WHT-%d", index)] = []models.HTTPRequestRule{
					*httpRequest1,
				}
			} else {
				c.cfg.HTTPRequests[fmt.Sprintf("WHT-%d", index)] = []models.HTTPRequestRule{
					*httpRequest2, //reverse order
					*httpRequest1,
				}
			}
		} else {
			c.cfg.HTTPRequests[fmt.Sprintf("WHT-%d", index)] = []models.HTTPRequestRule{}
		}
		c.cfg.HTTPRequestsStatus = MODIFIED
	case DELETED:
		c.cfg.HTTPRequests[fmt.Sprintf("WHT-%d", index)] = []models.HTTPRequestRule{}
	}
	//TODO BackendBalance proper usage
	balanceAlg := &models.BackendBalance{
		Algorithm: annBalanceAlg.Value,
	}
	if err != nil {
		log.Printf("%s, using %s \n", err, balanceAlg)
	}

	key := fmt.Sprintf("R%0006d", index)
	old, ok := c.cfg.UseBackendRules[key]
	if ok {
		if old.Backend != backendName || old.Host != rule.Host || old.Path != path.Path {
			c.cfg.UseBackendRules[key] = BackendSwitchingRule{
				Host:    rule.Host,
				Path:    path.Path,
				Backend: backendName,
			}
			c.cfg.UseBackendRulesStatus = MODIFIED
		}
	} else {
		c.cfg.UseBackendRules[key] = BackendSwitchingRule{
			Host:    rule.Host,
			Path:    path.Path,
			Backend: backendName,
		}
		c.cfg.UseBackendRulesStatus = MODIFIED
	}

	switch service.Status {
	case ADDED:
		if numberOfTimesBackendUsed := backendsUsed[backendName]; numberOfTimesBackendUsed < 2 {
			backend := &models.Backend{
				Balance: balanceAlg,
				Name:    backendName,
				Mode:    "http",
			}
			if annForwardedFor.Value == "enabled" { //disabled with anything else is ok
				forwardfor := "enabled"
				backend.Forwardfor = &models.BackendForwardfor{
					Enabled: &forwardfor,
				}
			}
			if err := nativeAPI.Configuration.CreateBackend(backend, transaction.ID, 0); err != nil {
				msg := err.Error()
				if !strings.Contains(msg, "Farm already exists") {
					return "", nil, nil, err
				}
			}
		}
	case MODIFIED:
		log.Println("so we have modified now")
	case DELETED:
		backendsUsed[backendName]--
		if err := nativeAPI.Configuration.DeleteBackend(backendName, transaction.ID, 0); err != nil {
			log.Println("DeleteBackend", err)
			return "", nil, nil, err
		}
		delete(c.cfg.UseBackendRules, fmt.Sprintf("R%0006d", index))
		c.cfg.UseBackendRulesStatus = MODIFIED
		return "", nil, service, nil
	}

	if annBalanceAlg.Status != "" || annForwardedFor.Status != "" {
		if err = c.handleBackendAnnotations(balanceAlg, annForwardedFor, backendName, transaction); err != nil {
			return "", nil, nil, err
		}
	}
	return backendName, selector, service, nil
}
