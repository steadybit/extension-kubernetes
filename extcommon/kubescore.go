package extcommon

import (
	"bytes"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-kubernetes/extconfig"
	"github.com/zegl/kube-score/config"
	ks "github.com/zegl/kube-score/domain"
	"github.com/zegl/kube-score/parser"
	"github.com/zegl/kube-score/score"
	"github.com/zegl/kube-score/scorecard"
	"io"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sJson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"strconv"
	"strings"
)

var (
	serializer = k8sJson.NewSerializerWithOptions(
		k8sJson.DefaultMetaFactory, nil, nil,
		k8sJson.SerializerOptions{
			Yaml:   true,
			Pretty: true,
			Strict: true,
		},
	)
)

type kubeScoreInput interface {
	runtime.Object
	GetName() string
	GetNamespace() string
}

func GetKubeScoreForDeployment(deployment *appsv1.Deployment, services []*corev1.Service, hpa *autoscalingv2.HorizontalPodAutoscaler) map[string][]string {
	inputs := make([]kubeScoreInput, 0)
	inputs = append(inputs, deployment)
	for _, service := range services {
		inputs = append(inputs, service)
	}
	if hpa != nil {
		inputs = append(inputs, hpa)
	}

	attributes := map[string][]string{}

	scores := getScores(inputs)
	addContainerResourceScores(scores, attributes)
	addContainerEphemeralStorageScores(scores, attributes)
	addContainerBasedScore(scores, attributes, "container-image-tag", "k8s.container.image.with-latest-tag")
	addContainerBasedScore(scores, attributes, "container-image-pull-policy", "k8s.container.image.without-image-pull-policy-always")
	addSimpleScore(scores, attributes, "deployment-has-host-podantiaffinity", "k8s.specification.has-host-podantiaffinity")
	addSimpleScore(scores, attributes, "deployment-targeted-by-hpa-does-not-have-replicas-configured", "k8s.specification.has-hpa-and-replicas-not-set")

	return attributes
}

func GetKubeScoreForDaemonSet(daemonSet *appsv1.DaemonSet, services []*corev1.Service) map[string][]string {
	inputs := make([]kubeScoreInput, 0)
	inputs = append(inputs, daemonSet)
	for _, service := range services {
		inputs = append(inputs, service)
	}

	attributes := map[string][]string{}

	scores := getScores(inputs)
	addContainerResourceScores(scores, attributes)
	addContainerEphemeralStorageScores(scores, attributes)
	addContainerBasedScore(scores, attributes, "container-image-tag", "k8s.container.image.with-latest-tag")
	addContainerBasedScore(scores, attributes, "container-image-pull-policy", "k8s.container.image.without-image-pull-policy-always")

	return attributes
}

func GetKubeScoreForStatefulSet(statefulSet *appsv1.StatefulSet, services []*corev1.Service) map[string][]string {
	inputs := make([]kubeScoreInput, 0)
	inputs = append(inputs, statefulSet)
	for _, service := range services {
		inputs = append(inputs, service)
	}
	attributes := map[string][]string{}

	scores := getScores(inputs)
	addContainerResourceScores(scores, attributes)
	addContainerEphemeralStorageScores(scores, attributes)
	addContainerBasedScore(scores, attributes, "container-image-tag", "k8s.container.image.with-latest-tag")
	addContainerBasedScore(scores, attributes, "container-image-pull-policy", "k8s.container.image.without-image-pull-policy-always")
	addSimpleScore(scores, attributes, "statefulset-has-host-podantiaffinity", "k8s.specification.has-host-podantiaffinity")

	return attributes
}

func addContainerResourceScores(scores []scorecard.TestScore, attributes map[string][]string) {
	score := getTestScore(scores, "container-resources")
	if score != nil {
		var containerNamesWithoutRequestCPU []string
		var containerNamesWithoutLimitCPU []string
		var containerNamesWithoutRequestMemory []string
		var containerNamesWithoutLimitMemory []string
		for _, comment := range score.Comments {
			if comment.Summary == "CPU request is not set" {
				containerNamesWithoutRequestCPU = append(containerNamesWithoutRequestCPU, comment.Path)
			} else if comment.Summary == "CPU limit is not set" {
				containerNamesWithoutLimitCPU = append(containerNamesWithoutLimitCPU, comment.Path)
			} else if comment.Summary == "Memory request is not set" {
				containerNamesWithoutRequestMemory = append(containerNamesWithoutRequestMemory, comment.Path)
			} else if comment.Summary == "Memory limit is not set" {
				containerNamesWithoutLimitMemory = append(containerNamesWithoutLimitMemory, comment.Path)
			}
		}
		if len(containerNamesWithoutRequestCPU) > 0 {
			attributes["k8s.container.spec.request.cpu.not-set"] = containerNamesWithoutRequestCPU
		}
		if len(containerNamesWithoutLimitCPU) > 0 {
			attributes["k8s.container.spec.limit.cpu.not-set"] = containerNamesWithoutLimitCPU
		}
		if len(containerNamesWithoutRequestMemory) > 0 {
			attributes["k8s.container.spec.request.memory.not-set"] = containerNamesWithoutRequestMemory
		}
		if len(containerNamesWithoutLimitMemory) > 0 {
			attributes["k8s.container.spec.limit.memory.not-set"] = containerNamesWithoutLimitMemory
		}
	}
}

func addContainerEphemeralStorageScores(scores []scorecard.TestScore, attributes map[string][]string) {
	score := getTestScore(scores, "container-ephemeral-storage-request-and-limit")
	if score != nil {
		var containerNamesWithoutRequestEphemeralStorage []string
		var containerNamesWithoutLimitEphemeralStorage []string
		for _, comment := range score.Comments {
			if comment.Summary == "Ephemeral Storage request is not set" {
				containerNamesWithoutRequestEphemeralStorage = append(containerNamesWithoutRequestEphemeralStorage, comment.Path)
			} else if comment.Summary == "Ephemeral Storage limit is not set" {
				containerNamesWithoutLimitEphemeralStorage = append(containerNamesWithoutLimitEphemeralStorage, comment.Path)
			}
		}
		if len(containerNamesWithoutRequestEphemeralStorage) > 0 {
			attributes["k8s.container.spec.request.ephemeral-storage.not-set"] = containerNamesWithoutRequestEphemeralStorage
		}
		if len(containerNamesWithoutLimitEphemeralStorage) > 0 {
			attributes["k8s.container.spec.limit.ephemeral-storage.not-set"] = containerNamesWithoutLimitEphemeralStorage
		}
	}
}

func addContainerBasedScore(scores []scorecard.TestScore, attributes map[string][]string, checkId string, attribute string) {
	score := getTestScore(scores, checkId)
	if score != nil {
		var containerNames []string
		for _, comment := range score.Comments {
			containerNames = append(containerNames, comment.Path)
		}
		if len(containerNames) > 0 {
			attributes[attribute] = containerNames
		}
	}
}

func addSimpleScore(scores []scorecard.TestScore, attributes map[string][]string, checkId string, attribute string) {
	score := getTestScore(scores, checkId)
	if score != nil {
		attributes[attribute] = []string{strconv.FormatBool(isCheckOk(score))}
	}
}

func getScores(inputs []kubeScoreInput) []scorecard.TestScore {
	manifests := prepareManifests(inputs)
	scoreCard := getKubeScoreCard(manifests)
	if scoreCard == nil {
		return []scorecard.TestScore{}
	}
	for _, scoredObject := range *scoreCard {
		if (scoredObject.ObjectMeta.Name == inputs[0].GetName()) && (scoredObject.ObjectMeta.Namespace == inputs[0].GetNamespace() && scoredObject.TypeMeta.Kind == inputs[0].GetObjectKind().GroupVersionKind().Kind) {
			return scoredObject.Checks
		}
	}
	return []scorecard.TestScore{}
}

func prepareManifests(objects []kubeScoreInput) []ks.NamedReader {
	manifests := make([]ks.NamedReader, 0, len(objects))
	for _, obj := range objects {
		manifestBuf := new(bytes.Buffer)
		err := serializer.Encode(obj, manifestBuf)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to marshal obj %s/%s", obj.GetName(), obj.GetNamespace())
		} else {
			manifests = append(manifests, inputReader{
				Reader: strings.NewReader(manifestBuf.String()),
				name:   fmt.Sprintf("%s/%s/%s", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetName(), obj.GetNamespace()),
			})
		}
	}
	return manifests
}

func getKubeScoreCard(manifests []ks.NamedReader) *scorecard.Scorecard {
	cnf := config.Configuration{
		AllFiles:                              manifests,
		VerboseOutput:                         0,
		IgnoreContainerCpuLimitRequirement:    extconfig.Config.AdviceIgnoreContainerCpuLimitRequirement,
		IgnoreContainerMemoryLimitRequirement: extconfig.Config.AdviceIgnoreContainerMemoryLimitRequirement,
		IgnoredTests:                          nil,
		EnabledOptionalTests:                  nil,
		UseIgnoreChecksAnnotation:             false,
		UseOptionalChecksAnnotation:           false,
	}

	p, err := parser.New()
	if err != nil {
		log.Error().Err(err).Msg("failed to create kubescore parser")
		return nil
	}
	parsedFiles, err := p.ParseFiles(cnf)
	if err != nil {
		log.Error().Err(err).Msg("failed to parse files")
		return nil
	}
	scoreCard, err := score.Score(parsedFiles, cnf)
	if err != nil {
		log.Error().Err(err).Msg("failed to run kubescore")
		return nil
	}
	return scoreCard
}

func getTestScore(scores []scorecard.TestScore, checkId string) *scorecard.TestScore {
	for _, check := range scores {
		if check.Check.ID == checkId && !check.Skipped {
			return &check
		}
	}
	return nil
}

func isCheckOk(score *scorecard.TestScore) bool {
	switch score.Grade {
	case scorecard.GradeCritical, scorecard.GradeWarning:
		return false
	case scorecard.GradeAlmostOK, scorecard.GradeAllOK:
		return true
	default:
		return false
	}
}

type inputReader struct {
	io.Reader
	name string
}

func (p inputReader) Name() string {
	return p.name
}
