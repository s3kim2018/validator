package resources

import (
	"context"
	"fmt"

	"github.com/michaelhenkel/validator/graph"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	introspectcontrolv1alpha1 "github.com/michaelhenkel/introspect/pkg/apis/control/v1alpha1"
)

const (
	introspectPort string = "8083"
)

type BGPNeighborNode struct {
	Resource      introspectcontrolv1alpha1.BgpNeighborResp
	EdgeLabels    []graph.EdgeLabel
	EdgeSelectors []graph.EdgeSelector
}

func (r *BGPNeighborNode) Convert(in graph.NodeInterface) error {
	_, ok := in.(*BGPNeighborNode)
	if !ok {
		return fmt.Errorf("not a BGPNeighborNode resource")
	}
	graph.Convert(in, r)
	return nil
}

func (r *BGPNeighborNode) AdderFunc() func(g *graph.Graph) ([]graph.NodeInterface, error) {
	return r.Adder
}

func (r *BGPNeighborNode) Name() string {
	return r.Resource.Peer.Text
}

func (r *BGPNeighborNode) Type() graph.NodeType {
	return graph.BGPNeighbor
}

func (r *BGPNeighborNode) GetEdgeLabels() []graph.EdgeLabel {
	return r.EdgeLabels
}

func (r *BGPNeighborNode) GetEdgeSelectors() []graph.EdgeSelector {
	return r.EdgeSelectors
}

func (r *BGPNeighborNode) Adder(g *graph.Graph) ([]graph.NodeInterface, error) {
	var graphNodeList []graph.NodeInterface
	var filterMap = make(map[string]struct{})

	controlList, err := g.ClientConfig.DeployerControlV1.Controls("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, controlResource := range controlList.Items {
		filterMap[controlResource.Name] = struct{}{}
	}

	for controlFilter := range filterMap {
		opts := metav1.ListOptions{
			LabelSelector: fmt.Sprintf("app=%s", controlFilter),
		}
		podList, err := g.ClientConfig.CoreV1.Pods("").List(context.Background(), opts)
		if err != nil {
			return nil, err
		}
		for _, podResource := range podList.Items {
			bgpNeighborList, err := g.ClientConfig.IntrospectClientV1.Control(podResource.Status.PodIP + ":" + introspectPort).BgpNeighbors().List(context.Background())
			if err != nil {
				return nil, err
			}
			for _, resource := range bgpNeighborList.Neighbors.List.BgpNeighborResp {
				r.Resource = resource
				resourceNode := &BGPNeighborNode{
					Resource: resource,
					EdgeSelectors: []graph.EdgeSelector{{
						NodeType: graph.VirtualRouter,
						MatchValues: []graph.MatchValue{{
							Value: map[string]string{"VirtualRouterIP": resource.PeerAddress.Text},
						}},
					}},
					EdgeLabels: []graph.EdgeLabel{{
						Value: map[string]string{"BGPRouterNeighborLocalIP": resource.LocalAddress.Text},
					}},
				}
				graphNodeList = append(graphNodeList, resourceNode)
			}
		}
	}

	return graphNodeList, nil
}
