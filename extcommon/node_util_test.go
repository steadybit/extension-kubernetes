package extcommon

import (
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestAddNodeLabels(t *testing.T) {
	type args struct {
		nodes      []*corev1.Node
		nodeName   string
		attributes map[string][]string
	}
	tests := []struct {
		name string
		args args
		want map[string][]string
	}{
		{
			name: "should add label to attributes",
			args: args{
				nodes: []*corev1.Node{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "node1",
							Labels: map[string]string{
								"topology.kubernetes.io/region": "eu-central-1",
							},
						},
					},
				},
				nodeName:   "node1",
				attributes: map[string][]string{},
			},

			want: map[string][]string{
				"k8s.label.topology.kubernetes.io/region": {"eu-central-1"},
			},
		},
		{
			name: "should append label to existing attributes",
			args: args{
				nodes: []*corev1.Node{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "node1",
							Labels: map[string]string{
								"topology.kubernetes.io/region": "eu-central-1",
							},
						},
					},
				},
				nodeName: "node1",
				attributes: map[string][]string{
					"k8s.label.topology.kubernetes.io/region": {"us-central-1"},
				},
			},

			want: map[string][]string{
				"k8s.label.topology.kubernetes.io/region": {"us-central-1", "eu-central-1"},
			},
		},
		{
			name: "should not append same label twice",
			args: args{
				nodes: []*corev1.Node{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "node1",
							Labels: map[string]string{
								"topology.kubernetes.io/region": "eu-central-1",
							},
						},
					},
				},
				nodeName: "node1",
				attributes: map[string][]string{
					"k8s.label.topology.kubernetes.io/region": {"eu-central-1"},
				},
			},

			want: map[string][]string{
				"k8s.label.topology.kubernetes.io/region": {"eu-central-1"},
			},
		},
		{
			name: "should not add filtered label",
			args: args{
				nodes: []*corev1.Node{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "node1",
							Labels: map[string]string{
								"topology.kubernetes.io/ignore-me": "foobar",
							},
						},
					},
				},
				nodeName:   "node1",
				attributes: map[string][]string{},
			},

			want: map[string][]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, AddNodeLabels(tt.args.nodes, tt.args.nodeName, tt.args.attributes), "AddNodeLabels(%v, %v, %v)", tt.args.nodes, tt.args.nodeName, tt.args.attributes)
		})
	}
}
