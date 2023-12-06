package extcommon

import (
	"bytes"
	"github.com/rs/zerolog/log"
	"github.com/zegl/kube-score/config"
	ks "github.com/zegl/kube-score/domain"
	"github.com/zegl/kube-score/parser"
	"github.com/zegl/kube-score/score"
	"github.com/zegl/kube-score/scorecard"
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
	checks = map[string]string{"deployment-has-host-podantiaffinity": "k8s.specification.has-host-podantiaffinity"}
)

func getKubeScore(manifest string) (*scorecard.Scorecard, error) {

	reader := &inputReader{
		Reader: strings.NewReader(manifest),
	}

	cnf := config.Configuration{
		AllFiles:                              []ks.NamedReader{reader},
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
		log.Error().Err(err).Msg("failed to create parser")
		return nil, err
	}
	parsedFiles, err := p.ParseFiles(cnf)
	if err != nil {
		log.Error().Err(err).Msg("failed to parse files")
		return nil, err
	}

	scoreCard, err := score.Score(parsedFiles, cnf)
	if err != nil {
		return nil, err
	}
	return scoreCard, nil
}

func AddKubeScoreAttributes(obj runtime.Object, namespace string, name string, apiVersion string, kind string) map[string][]string {
	attributes := make(map[string][]string)
	manifestBuf := new(bytes.Buffer)
	err := serializer.Encode(obj, manifestBuf)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to marshal obj %s/%s", namespace, name)
	} else {

		manifest := "apiVersion: " + apiVersion + "\nkind: " + kind + "\n" + manifestBuf.String()
		scoreCard, err := getKubeScore(manifest)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to get kube-score for obj %s/%s", namespace, name)
		} else {
			for _, scoredObject := range *scoreCard {
				for _, check := range scoredObject.Checks {
					if checks[check.Check.ID] != "" {
						attributes[checks[check.Check.ID]] = []string{strconv.FormatBool(gradePassedCheck(check))}
					}
				}
			}
		}
	}
	return attributes
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
	*strings.Reader
}

func (inputReader) Name() string {
	return "input"
}