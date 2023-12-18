package extcommon

import (
	"bytes"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/zegl/kube-score/config"
	ks "github.com/zegl/kube-score/domain"
	"github.com/zegl/kube-score/parser"
	"github.com/zegl/kube-score/score"
	"github.com/zegl/kube-score/scorecard"
	"io"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sJson "k8s.io/apimachinery/pkg/runtime/serializer/json"
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

func GetKubeScoreForDeployment(deployment *appsv1.Deployment, services []*corev1.Service, hpa *autoscalingv1.HorizontalPodAutoscaler) *scorecard.ScoredObject {
	inputs := make([]kubeScoreInput, 0)
	inputs = append(inputs, deployment)
	for _, service := range services {
		inputs = append(inputs, service)
	}
	if hpa != nil {
		inputs = append(inputs, hpa)
	}
	manifests := prepareManifests(inputs)
	scoreCard := getKubeScoreCard(manifests)
	return getScoredObject(scoreCard, deployment)
}
func GetKubeScoreForDaemonSet(daemonSet *appsv1.DaemonSet, services []*corev1.Service) *scorecard.ScoredObject {
	inputs := make([]kubeScoreInput, 0)
	inputs = append(inputs, daemonSet)
	for _, service := range services {
		inputs = append(inputs, service)
	}
	manifests := prepareManifests(inputs)
	scoreCard := getKubeScoreCard(manifests)
	return getScoredObject(scoreCard, daemonSet)
}
func GetKubeScoreForStatefulSet(statefulSet *appsv1.StatefulSet, services []*corev1.Service) *scorecard.ScoredObject {
	inputs := make([]kubeScoreInput, 0)
	inputs = append(inputs, statefulSet)
	for _, service := range services {
		inputs = append(inputs, service)
	}
	manifests := prepareManifests(inputs)
	scoreCard := getKubeScoreCard(manifests)
	return getScoredObject(scoreCard, statefulSet)
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
		IgnoreContainerCpuLimitRequirement:    true,
		IgnoreContainerMemoryLimitRequirement: true,
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

func getScoredObject(scorecard *scorecard.Scorecard, object kubeScoreInput) *scorecard.ScoredObject {
	if scorecard == nil {
		return nil
	}
	for _, scoredObject := range *scorecard {
		if (scoredObject.ObjectMeta.Name == object.GetName()) && (scoredObject.ObjectMeta.Namespace == object.GetNamespace() && scoredObject.TypeMeta.Kind == object.GetObjectKind().GroupVersionKind().Kind) {
			return scoredObject
		}
	}
	return nil
}

func HasCheckResult(scoreCard *scorecard.ScoredObject, checkId string) bool {
	if scoreCard == nil {
		return false
	}
	for _, check := range scoreCard.Checks {
		if check.Check.ID == checkId && !check.Skipped {
			return true
		}
	}
	return false
}

func IsCheckOk(scoreCard *scorecard.ScoredObject, checkId string) bool {
	if scoreCard == nil {
		return false
	}
	for _, check := range scoreCard.Checks {
		if check.Check.ID == checkId && !check.Skipped && gradePassedCheck(check) {
			return gradePassedCheck(check)
		}
	}
	return false
}

func gradePassedCheck(check scorecard.TestScore) bool {
	switch check.Grade {
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
