package extingress

import (
	"context"
	"fmt"
	networkingv1 "k8s.io/api/networking/v1"
	"strings"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-kit/extutil"
)

// HAProxyDelayTrafficState contains state data for the HAProxy delay traffic action
type HAProxyDelayTrafficState struct {
	HAProxyBaseState
	ResponseDelay        int
	ConditionPathPattern string
	ConditionHttpMethod  string
	ConditionHttpHeader  map[string]string
	AnnotationConfig     string
}

// NewHAProxyDelayTrafficAction creates a new delay traffic action
func NewHAProxyDelayTrafficAction() action_kit_sdk.Action[HAProxyDelayTrafficState] {
	return &HAProxyDelayTrafficAction{}
}

// HAProxyDelayTrafficAction implements the delay traffic action
type HAProxyDelayTrafficAction struct{}

// NewEmptyState creates an empty state object
func (a *HAProxyDelayTrafficAction) NewEmptyState() HAProxyDelayTrafficState {
	return HAProxyDelayTrafficState{}
}

// Describe returns the action description for the HAProxy delay traffic action
func (a *HAProxyDelayTrafficAction) Describe() action_kit_api.ActionDescription {
	desc := getCommonActionDescription(
		HAProxyDelayTrafficActionId,
		"HAProxy Delay Traffic",
		"Delay traffic by adding a response delay for requests matching specific conditions.",
		"data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M10.6182%2021.6082L10.752%2022.1219H11.0391L11.0908%2022.0809L11.1006%2022.0701C11.152%2022.1215%2011.2035%2022.1736%2011.2549%2022.2147L11.1729%2022.2762V22.8719H10.4121V22.1111H10.5459L10.3203%2021.2791C10.4128%2021.3921%2010.5155%2021.4952%2010.6182%2021.6082ZM11.1113%201.84747L11.7178%202.31036L12.4268%201.74493V1.22052H13.1865V1.98126H13.0225L13.3408%203.08087H13.8135L14.4512%202.28986V1.61115H15.2109V2.37189H15.1699L15.5713%203.03986L16.3721%202.82404V2.45392H17.1328V3.21466H16.8242L16.7324%204.25177H17.2256V4.45782L18.1299%203.98517V3.61505H18.8896V4.37579H18.6533L18.7354%205.12579L19.5576%205.24884V5.07404H20.3184V5.82404H19.8252L19.3623%206.65704H19.5273V7.15997L20.709%207.06818V6.78009H21.4688V7.53009H21.0273L20.7295%208.23907L21.5518%208.70197V8.67072H22.3115V9.43146H21.5518V9.39044L21.4893%209.43146C21.4278%209.38022%2021.3769%209.33867%2021.3154%209.29767L21.5518%209.14337V8.93829L20.6475%208.42462L20.4932%208.77325C20.4315%208.74243%2020.3693%208.71228%2020.3076%208.68146L20.4619%208.31134L19.5371%207.78693V7.93048H18.6328L18.5713%208.02325C18.4686%207.99244%2018.3557%207.9613%2018.2529%207.93048H18.2324V7.36603L16.999%207.21173L17.0605%207.78693C16.958%207.76646%2016.8555%207.76642%2016.7529%207.76642L16.6904%207.17072L14.9951%206.9549V7.64337H14.4307L14.4609%208.04376C14.2554%208.09514%2014.0497%208.15738%2013.8545%208.22931L13.8037%207.64337H13.6084L13.1562%208.15704V8.51642C12.1287%208.9891%2011.2037%209.66795%2010.4639%2010.5106V9.69806L9.56934%209.80157L9.47754%2010.7361H10.2578C9.61056%2011.517%209.10713%2012.4213%208.77832%2013.3973H7.9873L7.46289%2013.8494V14.0652L8.5625%2014.1785C8.51112%2014.3738%208.48005%2014.5795%208.44922%2014.785L7.45312%2014.6824V15.2264H6.75391L6.92871%2016.8504L8.42871%2017.0145C8.44923%2017.1273%208.45998%2017.2302%208.48047%2017.3328L6.96973%2017.1688L7.11426%2018.5047H7.6582L8.57324%2017.8367C8.61435%2017.9909%208.65519%2018.1455%208.69629%2018.2996H8.63477V18.1756L7.6582%2018.8748V19.7791H7.51465L8.12109%2020.8065L9.50781%2020.3856V20.2156C9.66367%2020.4659%209.82693%2020.7082%2010.001%2020.9402H9.75488L9.09668%2021.783V22.492H8.33691V21.7313H8.3877L7.99805%2021.0633L7.22656%2021.2996V21.7H6.4668V20.9402H6.74414L6.83594%2019.7889H6.34277V19.6141L5.49023%2020.0867V20.5086H4.72949V19.7479H4.9248L4.84277%2018.9568L4.06152%2018.8543V19.0701H3.30176V18.3201H3.74316L4.25684%2017.3846H4.07227V16.9227L2.90039%2017.035V17.3543H2.14062V16.6033H2.57227L2.89062%2015.8641L2.14062%2015.4422V15.4529H1.37988V14.7029H2.06836L3.38379%2013.8084V13.5008L1.75977%2013.2127V13.4285H1V12.6785H1.61621L2.28418%2012.0408L1.62695%2011.4354H1V10.6746H1.75977V10.8592L3.37305%2010.5516V10.2225L2.12988%209.40021V9.42072H1.37012V8.66095H2.12988V8.69122L2.90039%208.23907L2.58203%207.52032H2.14062V6.75958H2.90039V7.05743L4.04102%207.15021V6.6463H4.19531L3.72266%205.80353H3.28125V5.04376H4.04102V5.3006L4.83301%205.15607L4.93555%204.34454H4.70898V3.58478H5.46973V4.04669L6.32227%204.45782V4.23126H6.81543L6.74414%203.17365H6.44629V2.4129H7.20605V2.81329L7.95605%203.01935L8.35742%202.36115H8.33691V1.60138H9.09668V2.28986L9.72363%203.08087H10.1758L10.4941%201.96075H10.3506V1.20001H11.1113V1.84747ZM16.6396%209.59552C20.1538%209.59558%2022.9997%2012.4418%2023%2015.9559C23%2019.4702%2020.1539%2022.3171%2016.6396%2022.3172C13.1253%2022.3172%2010.2793%2019.4702%2010.2793%2015.9559C10.2796%2012.4418%2013.1255%209.59552%2016.6396%209.59552ZM8.23438%2020.991L8.66504%2021.7215H8.91211V21.7313L9.52832%2020.9402H9.51855V20.6014L8.23438%2020.991ZM6.98047%2020.9402H7.25781V21.074L7.26758%2021.0838L7.92578%2020.8787L7.27832%2019.7889H7.07227L6.98047%2020.9402ZM16.6396%2011.0135C13.9064%2011.0135%2011.6975%2013.2227%2011.6973%2015.9559C11.6973%2018.6892%2013.9063%2020.8992%2016.6396%2020.8992C19.373%2020.8992%2021.582%2018.6892%2021.582%2015.9559C21.5818%2013.2228%2019.3728%2011.0135%2016.6396%2011.0135ZM9.50781%2020.2156C9.49813%2020.2001%209.48715%2020.1854%209.47754%2020.1697L9.50781%2020.159V20.2156ZM5.16113%2019.7586H5.51074V19.8504L6.36328%2019.3885V19.1727L5.0791%2018.9979L5.16113%2019.7586ZM5.37695%2017.3953H4.9043L5.05859%2018.782L6.36328%2018.9568V18.5047H6.79492L6.6416%2017.1277L5.37695%2016.994V17.3953ZM3.99023%2018.3201H4.08203V18.6395L4.83301%2018.741L4.68848%2017.3846H4.50391L3.99023%2018.3201ZM2.80859%2016.6033H2.9209V16.8299L4.08203%2016.7166V16.5213L3.08594%2015.9666L2.80859%2016.6033ZM5.37695%2016.1209V16.6756L6.61035%2016.8094H6.62109L6.44629%2015.2166H6.01465L5.37695%2016.1209ZM16.6396%2012.7498C17.0301%2012.7499%2017.3486%2013.0693%2017.3486%2013.4598V15.2469H19.1367C19.527%2015.247%2019.8455%2015.5656%2019.8457%2015.9559C19.8457%2016.3463%2019.5271%2016.6648%2019.1367%2016.6649H16.6396C16.2492%2016.6649%2015.9307%2016.3464%2015.9307%2015.9559V13.4598C15.9307%2013.0693%2016.2492%2012.7498%2016.6396%2012.7498ZM4.68848%2014.1483H3.86719L3.16797%2015.7713L4.08203%2016.285V16.0897H5.01758L5.63379%2015.2166H5.54199V14.1375L4.68848%2013.8699V14.1483ZM2.16113%2014.8983V15.2059L2.99316%2015.6688L3.65137%2014.1483H3.41504V14.0555L2.16113%2014.8983ZM4.6582%2012.8426H4.69922V13.5418L5.55176%2013.8084V13.3152H5.68555V13.3055L5.02832%2012.2879L4.6582%2012.8426ZM5.20215%2012.0203L6.04492%2013.326H7.09375L7.67871%2012.8221V11.2908L7.09375%2010.7674H6.03516L5.20215%2012.0203ZM1.79102%2012.8123V12.9969L1.80078%2013.0076L3.41504%2013.2947V13.0584L2.46875%2012.1746L1.79102%2012.8123ZM2.61328%2012.0311H2.60254L3.49707%2012.8533H4.28809L4.84277%2012.0203L4.28809%2011.158H3.52734L2.61328%2012.0311ZM1.78027%2011.0652V11.2703L2.45898%2011.8973L3.39355%2011.0037V10.7567L1.78027%2011.0652ZM4.67871%2010.4998V11.1678H4.6582L5.02832%2011.743L5.6748%2010.7674L5.66504%2010.7772H5.53125V10.2635L4.67871%2010.4998ZM7.44238%2010.0272V10.2537L7.9668%2010.7156H8.85059L8.93262%209.8631L7.44238%2010.0272ZM3.20898%208.3006L3.89746%209.87286H4.69922V10.1707L5.55176%209.93439V8.86603H5.63379L4.97656%207.92072L4.99707%207.90997H4.09277V7.77716L3.20898%208.3006ZM2.16113%208.90704V9.13361L2.15039%209.14337L3.39355%209.96564V9.87286H3.65137L3.01367%208.40411L2.16113%208.90704ZM6.91895%207.17072H6.9082L6.7334%208.87677H7.44238V9.41095L8.99414%209.23615L9.14844%207.62286H8.62402V6.96466L6.91895%207.17072ZM9.63184%209.16388L10.4639%209.07111V8.18829L9.9707%207.62286H9.77539L9.63184%209.16388ZM5.35645%207.35529V7.92072H5.34668L6.00391%208.86603H6.42578L6.59961%207.20099L5.35645%207.35529ZM2.93164%207.49982H2.84375L3.10645%208.10529L4.07227%207.54083V7.33478L2.93164%207.242V7.49982ZM19.5166%207.31427V7.52032L20.5234%208.09552L20.7803%207.48907H20.6885V7.2215L19.5166%207.31427ZM10.5361%206.13263V7.32501L11.0186%207.88947H12.5596L13.084%207.29376H13.0947V6.16388L11.7998%205.17657L10.5361%206.13263ZM17.2461%205.50568L17.2354%205.51642H16.8037L16.958%206.89337L18.2217%207.04767V6.62579H18.6943L18.5508%205.26935L17.2461%205.07404V5.50568ZM5.06934%205.32111L4.9043%206.62579H5.36719V7.03693L6.6416%206.88263L6.78516%205.51642H6.36328V5.09454L5.06934%205.32111ZM14.9854%206.19415V6.6463L16.6494%206.85236L16.4961%205.51642H15.9512L14.9854%206.19415ZM6.95996%206.84161L8.63477%206.63654V6.21466L7.64844%205.51642H7.10254L6.95996%206.84161ZM4.08203%205.50568V5.79376H4L4.47266%206.63654H4.69922L4.85352%205.37189L4.08203%205.50568ZM18.9004%206.61603H19.1162L19.5889%205.78302H19.5576V5.4129L19.5479%205.40314L18.7559%205.28986L18.9004%206.61603ZM8.08984%203.26544L7.50391%204.242H7.64844V5.1463L8.63477%205.84454V5.7215H9.58984L9.85742%204.36505H9.49805V3.6463L8.10059%203.25568L8.08984%203.26544ZM14.0908%203.65607V4.37579H13.7314L13.999%205.75275H14.9951V5.81427L15.9609%205.13556V4.23126H16.085L15.5088%203.26544L14.0908%203.65607ZM13.4131%204.36505H12.8584L12.0459%204.98224L13.084%205.77325V5.73224H13.6797L13.4131%204.35529V4.36505ZM9.89844%205.71173L10.5361%205.73224L11.5322%204.9715L10.7412%204.36505H10.166V4.35529L9.89844%205.71173ZM5.51074%204.33478H5.19238L5.18164%204.34454L5.08984%205.11505L6.36328%204.88947V4.68341L5.51074%204.27325V4.33478ZM17.2461%204.6424V4.85822L18.5303%205.04376L18.458%204.32404H18.1396V4.18048L17.2461%204.6424ZM10.793%203.32697V4.01544L11.7998%204.78693L12.7959%204.02618V3.35822L11.7588%202.5672V2.55646L10.793%203.32697ZM7.25781%203.17365H6.99023L7.0625%204.23126H7.26758L7.89453%203.19415L7.25781%203.01935V3.17365ZM15.7148%203.20392L16.3213%204.2215H16.5469L16.6396%203.18341H16.3926V3.0213L15.7148%203.20392ZM8.63477%202.37189L8.20312%203.08087L9.49805%203.44025V3.09064L8.92285%202.36115H8.63477V2.37189ZM14.0703%203.10138V3.44025L15.3857%203.08087L14.9541%202.35138H14.6768L14.0703%203.10138ZM10.4326%203.09064H10.7725L11.5938%202.43341L10.9668%201.96075H10.752L10.4326%203.09064ZM11.9229%202.42267L12.7959%203.09064V3.07013H13.1562L12.8379%201.96075H12.5088V1.95001L11.9229%202.42267Z%22%20fill%3D%22currentColor%22%2F%3E%0A%3C%2Fsvg%3E%0A",
	)

	// Add delay-specific parameters
	desc.Parameters = append(desc.Parameters,
		[]action_kit_api.ActionParameter{
			{
				Name:  "-response-header-",
				Type:  action_kit_api.ActionParameterTypeHeader,
				Label: "Response",
			},
			{
				Name:         "responseDelay",
				Label:        "Delay",
				Description:  extutil.Ptr("The delay in milliseconds to add to matching requests"),
				Type:         action_kit_api.ActionParameterTypeDuration,
				DefaultValue: extutil.Ptr("500ms"),
				Required:     extutil.Ptr(true),
			},
		}...,
	)

	desc.Parameters = append(desc.Parameters, getConditionsParameters()...)

	return desc
}

// Prepare validates input parameters and prepares the state for execution
func (a *HAProxyDelayTrafficAction) Prepare(ctx context.Context, state *HAProxyDelayTrafficState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	ingress, err := prepareHAProxyAction(&state.HAProxyBaseState, request)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare HAProxy delay action: %w", err)
	}

	// Extract and validate delay parameter
	if delay, ok := request.Config["responseDelay"]; ok {
		switch v := delay.(type) {
		case float64:
			state.ResponseDelay = int(v)
		case int:
			state.ResponseDelay = v
		case string:
			return nil, fmt.Errorf("delay must be a number, got string: %s", v)
		default:
			return nil, fmt.Errorf("delay must be a number, got %T", v)
		}
	} else {
		return nil, fmt.Errorf("responseDelay parameter is required")
	}

	// Parse condition parameters
	state.ConditionPathPattern = extutil.ToString(request.Config["conditionPathPattern"])
	state.ConditionHttpMethod = extutil.ToString(request.Config["conditionHttpMethod"])

	if request.Config["conditionHttpHeader"] != nil {
		state.ConditionHttpHeader, err = extutil.ToKeyValue(request.Config, "conditionHttpHeader")
		if err != nil {
			return nil, fmt.Errorf("failed to parse HTTP header condition: %w", err)
		}
	}

	// Validate that at least one condition is specified
	if state.ConditionPathPattern == "" && state.ConditionHttpMethod == "" && len(state.ConditionHttpHeader) == 0 {
		return nil, fmt.Errorf("at least one condition (path, method, or header) is required")
	}

	// Check for conflicts with existing rules
	if err := checkForRuleConflicts(ingress, state); err != nil {
		return nil, err
	}

	// Build HAProxy configuration
	state.AnnotationConfig = buildDelayConfiguration(state)

	return nil, nil
}

// checkForRuleConflicts checks if the new rules would conflict with existing ones
func checkForRuleConflicts(ingress *networkingv1.Ingress, state *HAProxyDelayTrafficState) error {
	existingLines := strings.Split(ingress.Annotations[AnnotationKey], "\n")

	// Check path pattern conflicts
	if state.ConditionPathPattern != "" {
		for _, line := range existingLines {
			if strings.Contains(line, fmt.Sprintf("path_reg %s", state.ConditionPathPattern)) {
				return fmt.Errorf("a rule for path %s already exists", state.ConditionPathPattern)
			}
		}
	}

	// Check for existing delay rules
	for _, line := range existingLines {
		if strings.Contains(line, "tcp-request inspect-delay") {
			return fmt.Errorf("a delay rule already exists - cannot add another one")
		}
	}

	return nil
}

// buildDelayConfiguration creates the HAProxy configuration for traffic delay
func buildDelayConfiguration(state *HAProxyDelayTrafficState) string {
	var configBuilder strings.Builder
	configBuilder.WriteString(getStartMarker(state.ExecutionId) + "\n")

	// Add the delay inspection directive
	configBuilder.WriteString(fmt.Sprintf("tcp-request inspect-delay %dms\n", state.ResponseDelay))

	// Generate a unique ACL ID prefix based on the execution ID
	aclIdPrefix := strings.Replace(state.ExecutionId.String(), "-", "_", -1)

	// Define ACLs for each condition
	var aclDefinitions []string
	var invertedAclRefs []string

	// Add method condition if specified
	if state.ConditionHttpMethod != "" && state.ConditionHttpMethod != "*" {
		aclName := fmt.Sprintf("sb_method_%s", aclIdPrefix)
		aclDefinitions = append(aclDefinitions, fmt.Sprintf("acl %s method %s", aclName, state.ConditionHttpMethod))
		invertedAclRefs = append(invertedAclRefs, fmt.Sprintf("!%s", aclName))
	}

	// Add header conditions if specified
	if state.ConditionHttpHeader != nil {
		for headerName, headerValue := range state.ConditionHttpHeader {
			aclName := fmt.Sprintf("sb_hdr_%s_%s", strings.Replace(headerName, "-", "_", -1), aclIdPrefix)
			aclDefinitions = append(aclDefinitions, fmt.Sprintf("acl %s hdr(%s) -m reg %s", aclName, headerName, headerValue))
			invertedAclRefs = append(invertedAclRefs, fmt.Sprintf("!%s", aclName))
		}
	}

	// Add path pattern condition if specified
	if state.ConditionPathPattern != "" {
		aclName := fmt.Sprintf("sb_path_%s", aclIdPrefix)
		aclDefinitions = append(aclDefinitions, fmt.Sprintf("acl %s path_reg %s", aclName, state.ConditionPathPattern))
		invertedAclRefs = append(invertedAclRefs, fmt.Sprintf("!%s", aclName))
	}

	// Add all ACL definitions to the configuration
	for _, aclDef := range aclDefinitions {
		configBuilder.WriteString(aclDef + "\n")
	}

	// For delay, we need to accept packets that don't match our conditions immediately
	// HAProxy delays the request if it doesn't accept it immediately
	if len(invertedAclRefs) > 0 {
		invertedCondition := strings.Join(invertedAclRefs, " || ")
		configBuilder.WriteString(fmt.Sprintf("tcp-request content accept if WAIT_END || %s\n", invertedCondition))
	} else {
		// This should never happen due to the earlier check, but as a failsafe
		configBuilder.WriteString("tcp-request content accept if WAIT_END\n")
	}

	configBuilder.WriteString(getEndMarker(state.ExecutionId) + "\n")
	return configBuilder.String()
}

// Start applies the HAProxy configuration to begin delaying traffic
func (a *HAProxyDelayTrafficAction) Start(ctx context.Context, state *HAProxyDelayTrafficState) (*action_kit_api.StartResult, error) {
	if err := startHAProxyAction(&state.HAProxyBaseState, state.AnnotationConfig); err != nil {
		return nil, fmt.Errorf("failed to start HAProxy delay traffic action: %w", err)
	}
	return nil, nil
}

// Stop removes the HAProxy configuration to stop delaying traffic
func (a *HAProxyDelayTrafficAction) Stop(ctx context.Context, state *HAProxyDelayTrafficState) (*action_kit_api.StopResult, error) {
	if err := stopHAProxyAction(&state.HAProxyBaseState); err != nil {
		return nil, fmt.Errorf("failed to stop HAProxy delay traffic action: %w", err)
	}
	return nil, nil
}
