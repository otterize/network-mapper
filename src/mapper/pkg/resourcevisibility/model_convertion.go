package resourcevisibility

import (
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/otterize/nilable"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func convertPortProtocol(protocol corev1.Protocol) (*cloudclient.K8sPortProtocol, error) {
	if protocol == "" {
		return nil, errors.New("port protocol is empty")
	}
	var result cloudclient.K8sPortProtocol
	switch protocol {
	case corev1.ProtocolTCP:
		result = cloudclient.K8sPortProtocolTcp
	case corev1.ProtocolUDP:
		result = cloudclient.K8sPortProtocolUdp
	case corev1.ProtocolSCTP:
		result = cloudclient.K8sPortProtocolSctp
	default:
		return nil, errors.Errorf("unimplemented port protocol: %s", protocol)
	}
	return &result, nil
}

func convertIntOrString(port intstr.IntOrString) *cloudclient.IntOrStringInput {
	if lo.IsEmpty(port) {
		return nil
	}
	result := cloudclient.IntOrStringInput{
		IsInt: port.Type == intstr.Int,
	}
	if result.IsInt {
		result.IntVal = nilable.From(int(port.IntVal))
	} else {
		result.StrVal = nilable.From(port.StrVal)
	}
	return &result
}

func nilIfEmpty[T comparable](value T) nilable.Nilable[T] {
	if lo.IsEmpty(value) {
		return nilable.Nilable[T]{}
	}
	return nilable.From(value)
}

func convertServiceType(serviceType corev1.ServiceType) (nilable.Nilable[cloudclient.K8sServiceType], error) {
	if serviceType == "" {
		return nilable.Nilable[cloudclient.K8sServiceType]{}, nil
	}

	var result cloudclient.K8sServiceType
	switch serviceType {
	case corev1.ServiceTypeClusterIP:
		result = cloudclient.K8sServiceTypeClusterIp
	case corev1.ServiceTypeNodePort:
		result = cloudclient.K8sServiceTypeNodePort
	case corev1.ServiceTypeLoadBalancer:
		result = cloudclient.K8sServiceTypeLoadBalancer
	case corev1.ServiceTypeExternalName:
		result = cloudclient.K8sServiceTypeExternalName
	default:
		return nilable.Nilable[cloudclient.K8sServiceType]{}, errors.Errorf("unimplemented service type: %s", serviceType)
	}

	return nilable.From(result), nil
}

func convertSessionAffinity(sessionAffinity corev1.ServiceAffinity) (nilable.Nilable[cloudclient.SessionAffinity], error) {
	if sessionAffinity == "" {
		return nilable.Nilable[cloudclient.SessionAffinity]{}, nil
	}
	var result cloudclient.SessionAffinity
	switch sessionAffinity {
	case corev1.ServiceAffinityClientIP:
		result = cloudclient.SessionAffinityClientIp
	case corev1.ServiceAffinityNone:
		result = cloudclient.SessionAffinityNone
	default:
		return nilable.Nilable[cloudclient.SessionAffinity]{}, errors.Errorf("unimplemented session affinity: %s", sessionAffinity)
	}
	return nilable.From(result), nil
}

func convertExternalTrafficPolicy(externalTrafficPolicy corev1.ServiceExternalTrafficPolicyType) (nilable.Nilable[cloudclient.ServiceExternalTrafficPolicy], error) {
	if externalTrafficPolicy == "" {
		return nilable.Nilable[cloudclient.ServiceExternalTrafficPolicy]{}, nil
	}
	var result cloudclient.ServiceExternalTrafficPolicy
	switch externalTrafficPolicy {
	case corev1.ServiceExternalTrafficPolicyLocal:
		result = cloudclient.ServiceExternalTrafficPolicyLocal
	case corev1.ServiceExternalTrafficPolicyCluster:
		result = cloudclient.ServiceExternalTrafficPolicyCluster
	default:
		return nilable.Nilable[cloudclient.ServiceExternalTrafficPolicy]{}, errors.Errorf("unimplemented external traffic policy: %s", externalTrafficPolicy)
	}
	return nilable.From(result), nil
}

func convertSessionAffinityConfig(sessionAffinityConfig *corev1.SessionAffinityConfig) nilable.Nilable[cloudclient.SessionAffinityConfig] {
	var empty *cloudclient.SessionAffinityConfig
	if sessionAffinityConfig == nil || sessionAffinityConfig.ClientIP == nil || sessionAffinityConfig.ClientIP.TimeoutSeconds == nil {
		return nilable.FromPtr(empty)
	}

	return nilable.From(cloudclient.SessionAffinityConfig{
		ClientIP: nilable.From(cloudclient.ClientIPConfig{
			TimeoutSeconds: nilable.From(int(*sessionAffinityConfig.ClientIP.TimeoutSeconds)),
		}),
	})
}

func convertIpFamilyPolicy(ipFamilyPolicy *corev1.IPFamilyPolicyType) (nilable.Nilable[cloudclient.IpFamilyPolicy], error) {
	if ipFamilyPolicy == nil {
		return nilable.Nilable[cloudclient.IpFamilyPolicy]{}, nil
	}
	var result cloudclient.IpFamilyPolicy
	switch *ipFamilyPolicy {
	case corev1.IPFamilyPolicySingleStack:
		result = cloudclient.IpFamilyPolicySingleStack
	case corev1.IPFamilyPolicyPreferDualStack:
		result = cloudclient.IpFamilyPolicyPreferDualStack
	case corev1.IPFamilyPolicyRequireDualStack:
		result = cloudclient.IpFamilyPolicyRequireDualStack
	default:
		return nilable.Nilable[cloudclient.IpFamilyPolicy]{}, errors.Errorf("unimplemented ip family policy: %s", *ipFamilyPolicy)
	}
	return nilable.From(result), nil
}

func convertInternalTrafficPolicy(internalTrafficPolicy *corev1.ServiceInternalTrafficPolicyType) (nilable.Nilable[cloudclient.ServiceInternalTrafficPolicy], error) {
	if internalTrafficPolicy == nil {
		return nilable.Nilable[cloudclient.ServiceInternalTrafficPolicy]{}, nil
	}
	var result cloudclient.ServiceInternalTrafficPolicy
	switch *internalTrafficPolicy {
	case corev1.ServiceInternalTrafficPolicyCluster:
		result = cloudclient.ServiceInternalTrafficPolicyCluster
	case corev1.ServiceInternalTrafficPolicyLocal:
		result = cloudclient.ServiceInternalTrafficPolicyLocal
	default:
		return nilable.Nilable[cloudclient.ServiceInternalTrafficPolicy]{}, errors.Errorf("unimplemented internal traffic policy: %s", *internalTrafficPolicy)
	}
	return nilable.From(result), nil
}

func convertIpMode(ipMode *corev1.LoadBalancerIPMode) (nilable.Nilable[cloudclient.LoadBalancerIPMode], error) {
	if ipMode == nil {
		return nilable.Nilable[cloudclient.LoadBalancerIPMode]{}, nil
	}
	var result cloudclient.LoadBalancerIPMode
	switch *ipMode {
	case corev1.LoadBalancerIPModeVIP:
		result = cloudclient.LoadBalancerIPModeVip
	case corev1.LoadBalancerIPModeProxy:
		result = cloudclient.LoadBalancerIPModeProxy
	default:
		return nilable.Nilable[cloudclient.LoadBalancerIPMode]{}, errors.Errorf("unimplemented ip mode: %s", *ipMode)
	}
	return nilable.From(result), nil
}

func convertIPFamily(family corev1.IPFamily) (cloudclient.IPFamily, error) {
	switch family {
	case corev1.IPv4Protocol:
		return cloudclient.IPFamilyIpv4, nil
	case corev1.IPv6Protocol:
		return cloudclient.IPFamilyIpv6, nil
	case corev1.IPFamilyUnknown:
		return cloudclient.IPFamilyUnknown, nil
	default:
		return "", errors.Errorf("unimplemented ip family: %s", family)
	}
}

func convertServiceLoadBalancerPorts(ports []corev1.PortStatus) ([]cloudclient.PortStatusInput, error) {
	result := make([]cloudclient.PortStatusInput, 0)
	for _, port := range ports {
		if port.Protocol == "" {
			// Protocol is a required field in this object, the error log is here to make sure we don't miss it if it is
			// actually empty in real life somehow. Nevertheless, no need to return error here since the rest of the
			// object is still valid.
			logrus.Error("Service port status protocol is empty")
			continue
		}
		protocol, err := convertPortProtocol(port.Protocol)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		result = append(result, cloudclient.PortStatusInput{
			Port:     int(port.Port),
			Protocol: *protocol,
			Error:    nilable.FromPtr(port.Error),
		})
	}

	return result, nil
}

func convertIngressLoadBalancerPorts(ports []networkingv1.IngressPortStatus) ([]cloudclient.PortStatusInput, error) {
	result := make([]cloudclient.PortStatusInput, 0)
	for _, port := range ports {
		if port.Protocol == "" {
			// Protocol is a required field in this object, the error log is here to make sure we don't miss it if it is
			// actually empty in real life somehow. Nevertheless, no need to return error here since the rest of the
			// object is still valid.
			logrus.Error("Ingress port status protocol is empty")
			continue
		}
		protocol, err := convertPortProtocol(port.Protocol)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		result = append(result, cloudclient.PortStatusInput{
			Port:     int(port.Port),
			Protocol: *protocol,
			Error:    nilable.FromPtr(port.Error),
		})
	}

	return result, nil
}

func convertServiceLoadBalancerStatus(status corev1.ServiceStatus) (nilable.Nilable[cloudclient.K8sResourceServiceStatusInput], error) {
	loadBalancerStatus := status.LoadBalancer
	result := make([]cloudclient.K8sResourceServiceLoadBalancerIngressInput, 0)
	for _, lbIngress := range loadBalancerStatus.Ingress {
		ipMode, err := convertIpMode(lbIngress.IPMode)
		if err != nil {
			return nilable.Nilable[cloudclient.K8sResourceServiceStatusInput]{}, errors.Wrap(err)
		}

		ports, err := convertServiceLoadBalancerPorts(lbIngress.Ports)
		if err != nil {
			return nilable.Nilable[cloudclient.K8sResourceServiceStatusInput]{}, errors.Wrap(err)
		}

		ingressInput := cloudclient.K8sResourceServiceLoadBalancerIngressInput{
			Ip:       nilable.From(lbIngress.IP),
			Hostname: nilable.From(lbIngress.Hostname),
			IpMode:   ipMode,
			Ports:    ports,
		}
		result = append(result, ingressInput)
	}

	serviceStatusInput := cloudclient.K8sResourceServiceStatusInput{
		LoadBalancer: nilable.From(cloudclient.K8sResourceServiceLoadBalancerStatusInput{Ingress: result}),
	}

	return nilable.From(serviceStatusInput), nil
}

func convertIngressLoadBalancerStatus(status networkingv1.IngressStatus) (nilable.Nilable[cloudclient.K8sResourceIngressStatusInput], error) {
	ingressList := status.LoadBalancer.Ingress
	if len(ingressList) == 0 {
		return nilable.Nilable[cloudclient.K8sResourceIngressStatusInput]{}, nil
	}

	result := make([]cloudclient.K8sResourceLoadBalancerIngressInput, 0)
	for _, loadBalancer := range ingressList {
		ports, err := convertIngressLoadBalancerPorts(loadBalancer.Ports)
		if err != nil {
			return nilable.Nilable[cloudclient.K8sResourceIngressStatusInput]{}, errors.Wrap(err)
		}

		ingressInput := cloudclient.K8sResourceLoadBalancerIngressInput{
			Ip:       nilable.From(loadBalancer.IP),
			Hostname: nilable.From(loadBalancer.Hostname),
			Ports:    ports,
		}
		result = append(result, ingressInput)
	}

	ingressStatusInput := cloudclient.K8sResourceIngressStatusInput{
		LoadBalancer: result,
	}

	return nilable.From(ingressStatusInput), nil
}

func convertServiceResource(service corev1.Service) (cloudclient.K8sResourceServiceInput, error) {
	ports := make([]cloudclient.K8sServicePort, 0)
	for _, port := range service.Spec.Ports {
		protocol, err := convertPortProtocol(port.Protocol)
		if err != nil {
			return cloudclient.K8sResourceServiceInput{}, errors.Wrap(err)
		}

		ports = append(ports, cloudclient.K8sServicePort{
			Name:        nilable.From(port.Name),
			Protocol:    nilable.FromPtr(protocol),
			AppProtocol: nilable.FromPtr(port.AppProtocol),
			Port:        int(port.Port),
			TargetPort:  nilable.FromPtr(convertIntOrString(port.TargetPort)),
			NodePort:    nilable.From(int(port.NodePort)),
		})
	}

	selector := make([]cloudclient.SelectorKeyValueInput, 0)
	for key, value := range service.Spec.Selector {
		selector = append(selector, cloudclient.SelectorKeyValueInput{
			Key:   nilable.From(key),
			Value: nilable.From(value),
		})
	}

	serviceType, err := convertServiceType(service.Spec.Type)
	if err != nil {
		return cloudclient.K8sResourceServiceInput{}, errors.Wrap(err)
	}

	sessionAffinity, err := convertSessionAffinity(service.Spec.SessionAffinity)
	if err != nil {
		return cloudclient.K8sResourceServiceInput{}, errors.Wrap(err)
	}

	externalTrafficPolicy, err := convertExternalTrafficPolicy(service.Spec.ExternalTrafficPolicy)
	if err != nil {
		return cloudclient.K8sResourceServiceInput{}, errors.Wrap(err)
	}

	sessionAffinityConfig := convertSessionAffinityConfig(service.Spec.SessionAffinityConfig)
	ipFamilies := make([]cloudclient.IPFamily, 0)
	for _, family := range service.Spec.IPFamilies {
		ipFamily, err := convertIPFamily(family)
		if err != nil {
			return cloudclient.K8sResourceServiceInput{}, errors.Wrap(err)
		}
		ipFamilies = append(ipFamilies, ipFamily)
	}

	ipFamilyPolicy, err := convertIpFamilyPolicy(service.Spec.IPFamilyPolicy)
	if err != nil {
		return cloudclient.K8sResourceServiceInput{}, errors.Wrap(err)
	}

	internalTrafficPolicy, err := convertInternalTrafficPolicy(service.Spec.InternalTrafficPolicy)
	if err != nil {
		return cloudclient.K8sResourceServiceInput{}, errors.Wrap(err)
	}

	status, err := convertServiceLoadBalancerStatus(service.Status)
	if err != nil {
		return cloudclient.K8sResourceServiceInput{}, errors.Wrap(err)
	}

	spec := cloudclient.K8sResourceServiceSpecInput{
		Ports:                         ports,
		Selector:                      selector,
		ClusterIP:                     nilIfEmpty(service.Spec.ClusterIP),
		ClusterIPs:                    service.Spec.ClusterIPs,
		Type:                          serviceType,
		ExternalIPs:                   service.Spec.ExternalIPs,
		SessionAffinity:               sessionAffinity,
		LoadBalancerIP:                nilIfEmpty(service.Spec.LoadBalancerIP),
		LoadBalancerSourceRanges:      service.Spec.LoadBalancerSourceRanges,
		ExternalName:                  nilIfEmpty(service.Spec.ExternalName),
		ExternalTrafficPolicy:         externalTrafficPolicy,
		HealthCheckNodePort:           nilIfEmpty(int(service.Spec.HealthCheckNodePort)),
		PublishNotReadyAddresses:      nilIfEmpty(service.Spec.PublishNotReadyAddresses),
		SessionAffinityConfig:         sessionAffinityConfig,
		IpFamilies:                    ipFamilies,
		IpFamilyPolicy:                ipFamilyPolicy,
		AllocateLoadBalancerNodePorts: nilable.FromPtr(service.Spec.AllocateLoadBalancerNodePorts),
		LoadBalancerClass:             nilable.FromPtr(service.Spec.LoadBalancerClass),
		InternalTrafficPolicy:         internalTrafficPolicy,
	}

	input := cloudclient.K8sResourceServiceInput{
		Spec:   spec,
		Status: status,
	}

	return input, nil
}

func convertIngressBackend(backend *networkingv1.IngressBackend) (nilable.Nilable[cloudclient.K8sIngressBackendInput], error) {
	if backend == nil {
		return nilable.Nilable[cloudclient.K8sIngressBackendInput]{}, nil
	}

	if backend.Service == nil && backend.Resource == nil {
		return nilable.Nilable[cloudclient.K8sIngressBackendInput]{}, errors.Errorf("both service and resource are nil in ingress backend")
	}

	result := cloudclient.K8sIngressBackendInput{}
	if backend.Service != nil {
		port := cloudclient.ServiceBackendPortInput{}
		if backend.Service.Port.Name != "" {
			port.Name = nilable.From(backend.Service.Port.Name)
		} else {
			port.Number = nilable.From(int(backend.Service.Port.Number))
		}
		service := cloudclient.K8sIngressServiceBackendInput{
			Name: backend.Service.Name,
			Port: port,
		}
		result.Service = nilable.From(service)
	} else {
		resource := cloudclient.K8sIngressResourceBackendInput{
			Kind: backend.Resource.Kind,
			Name: backend.Resource.Name,
		}
		if backend.Resource.APIGroup != nil {
			resource.ApiGroup = nilable.From(*backend.Resource.APIGroup)
		}

		result.Resource = nilable.From(resource)
	}

	return nilable.From(result), nil
}

func convertIngressTLS(tls []networkingv1.IngressTLS) ([]cloudclient.K8sIngressTLSInput, error) {
	result := make([]cloudclient.K8sIngressTLSInput, 0)
	for _, tlsItem := range tls {
		hosts := make([]string, 0)
		if len(tlsItem.Hosts) > 0 {
			hosts = tlsItem.Hosts
		}
		secretName := nilIfEmpty(tlsItem.SecretName)
		result = append(result, cloudclient.K8sIngressTLSInput{
			Hosts:      hosts,
			SecretName: secretName,
		})
	}

	return result, nil
}

func convertPathType(pathType *networkingv1.PathType) (nilable.Nilable[cloudclient.PathType], error) {
	if pathType == nil {
		return nilable.Nilable[cloudclient.PathType]{}, nil
	}
	var result cloudclient.PathType
	switch *pathType {
	case networkingv1.PathTypeExact:
		result = cloudclient.PathTypeExact
	case networkingv1.PathTypePrefix:
		result = cloudclient.PathTypePrefix
	case networkingv1.PathTypeImplementationSpecific:
		result = cloudclient.PathTypeImplementationSpecific
	default:
		return nilable.Nilable[cloudclient.PathType]{}, errors.Errorf("unimplemented path type: %s", *pathType)
	}
	return nilable.From(result), nil
}

func convertIngressRulePaths(httpRule *networkingv1.HTTPIngressRuleValue) ([]cloudclient.K8sIngressHttpPathInput, error) {
	result := make([]cloudclient.K8sIngressHttpPathInput, 0)
	if httpRule == nil {
		return result, nil
	}
	paths := httpRule.Paths
	for _, path := range paths {
		pathType, err := convertPathType(path.PathType)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		backend, err := convertIngressBackend(&path.Backend)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		result = append(result, cloudclient.K8sIngressHttpPathInput{
			Path:     nilIfEmpty(path.Path),
			PathType: pathType,
			Backend:  backend.Item,
		})
	}
	return result, nil

}

func convertIngressRules(rules []networkingv1.IngressRule) ([]cloudclient.K8sIngressRuleInput, error) {
	result := make([]cloudclient.K8sIngressRuleInput, 0)
	for _, rule := range rules {
		paths, _ := convertIngressRulePaths(rule.HTTP)
		result = append(result, cloudclient.K8sIngressRuleInput{
			Host:      nilIfEmpty(rule.Host),
			HttpPaths: paths,
		})
	}
	return result, nil
}

func convertIngressResource(ingress networkingv1.Ingress) (cloudclient.K8sResourceIngressInput, error) {
	defaultBackend, err := convertIngressBackend(ingress.Spec.DefaultBackend)
	if err != nil {
		return cloudclient.K8sResourceIngressInput{}, errors.Wrap(err)
	}

	tls, err := convertIngressTLS(ingress.Spec.TLS)
	if err != nil {
		return cloudclient.K8sResourceIngressInput{}, errors.Wrap(err)
	}

	rules, err := convertIngressRules(ingress.Spec.Rules)
	if err != nil {
		return cloudclient.K8sResourceIngressInput{}, errors.Wrap(err)
	}

	spec := cloudclient.K8sResourceIngressSpecInput{
		IngressClassName: nilable.FromPtr(ingress.Spec.IngressClassName),
		DefaultBackend:   defaultBackend,
		Tls:              tls,
		Rules:            rules,
	}

	status, err := convertIngressLoadBalancerStatus(ingress.Status)
	if err != nil {
		return cloudclient.K8sResourceIngressInput{}, errors.Wrap(err)
	}

	input := cloudclient.K8sResourceIngressInput{
		Spec:   spec,
		Status: status,
	}

	return input, nil
}
