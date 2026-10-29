package metrics

import (
	"context"
	"fmt"
	"net"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// EndpointResolver lists Ready pod IPs for a Service via EndpointSlices.
type EndpointResolver struct {
	Client kubernetes.Interface
}

// ScrapeURLs returns http://ip:port/path for each Ready endpoint.
func (r *EndpointResolver) ScrapeURLs(ctx context.Context, namespace, service, port, path string) ([]string, error) {
	if r.Client == nil {
		return nil, fmt.Errorf("kubernetes client is nil")
	}
	slices, err := r.Client.DiscoveryV1().EndpointSlices(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("kubernetes.io/service-name=%s", service),
	})
	if err != nil {
		return nil, fmt.Errorf("list EndpointSlices for %s/%s: %w", namespace, service, err)
	}

	urls := make([]string, 0)
	seen := map[string]struct{}{}
	for _, slice := range slices.Items {
		for _, ep := range slice.Endpoints {
			if ep.Conditions.Ready != nil && !*ep.Conditions.Ready {
				continue
			}
			for _, addr := range ep.Addresses {
				key := net.JoinHostPort(addr, port)
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				urls = append(urls, fmt.Sprintf("http://%s%s", key, path))
			}
		}
	}
	return urls, nil
}
