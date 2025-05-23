package extingress

import (
	"context"
	"fmt"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-kit/extutil"
	"strings"
)

// HAProxyBlockTrafficState extends base state with block-specific fields
type HAProxyBlockTrafficState struct {
	HAProxyBaseState
	ResponseStatusCode   int
	ConditionPathPattern string
	ConditionHttpMethod  string
	ConditionHttpHeader  map[string]string
	//ConditionDownstreamServiceName string
	AnnotationConfig string
}

func NewHAProxyBlockTrafficAction() action_kit_sdk.Action[HAProxyBlockTrafficState] {
	return &HAProxyBlockTrafficAction{}
}

type HAProxyBlockTrafficAction struct{}

func (a *HAProxyBlockTrafficAction) NewEmptyState() HAProxyBlockTrafficState {
	return HAProxyBlockTrafficState{}
}

func (a *HAProxyBlockTrafficAction) Describe() action_kit_api.ActionDescription {
	desc := getCommonActionDescription(
		HAProxyBlockTrafficActionId,
		"HAProxy Block Traffic",
		"Block traffic by returning a custom HTTP status code for requests matching specific paths.",
		"data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M10.6455%2021.6649L10.7793%2022.1805H11.0674L11.1191%2022.1385L11.1299%2022.1287C11.1813%2022.1801%2011.2328%2022.2321%2011.2842%2022.2733L11.2012%2022.3348V22.9324H10.4395V22.1697H10.5732L10.3467%2021.3348C10.4394%2021.4481%2010.5425%2021.5515%2010.6455%2021.6649ZM11.1396%201.84943L11.748%202.31329L12.459%201.74689V1.22052H13.2217V1.98322H13.0566L13.376%203.08575H13.8496L14.4893%202.29279V1.61212H15.251V2.37482H15.21L15.6123%203.04474L16.416%202.82794V2.45782H17.1787V3.21954H16.8691L16.7764%204.26056H17.2715V4.46661L18.1777%203.99298V3.62189H18.9404V4.38458H18.7031L18.7861%205.13654L19.6104%205.26056V5.08478H20.373V5.83771H19.8779L19.4141%206.67169H19.5791V7.17657L20.7646%207.0838V6.79572H21.5264V7.54767H21.084L20.7852%208.25861L21.6094%208.72247V8.69122H22.3721V9.45392H21.6094V9.4129L21.5469%209.45392C21.4852%209.40255%2021.4338%209.36126%2021.3721%209.32013L21.6094%209.16583V8.95978L20.7021%208.44415L20.5479%208.79474L20.3623%208.70197L20.5166%208.33087L19.5898%207.80548V7.95001H18.6826L18.6211%208.04279C18.5182%208.01191%2018.4047%207.98089%2018.3018%207.95001H18.2812V7.38263L17.0439%207.22833L17.1064%207.80548C17.0034%207.78487%2016.8999%207.78497%2016.7969%207.78497L16.7354%207.18732L15.0352%206.97052V7.66095H14.4678L14.499%208.06329C14.2931%208.11478%2014.0873%208.17677%2013.8916%208.24884L13.8398%207.66095H13.6436L13.1904%208.17657V8.53693C12.16%209.01093%2011.2322%209.69099%2010.4902%2010.5359V9.72247L9.59375%209.82501L9.50195%2010.7625H10.2842C9.63506%2011.5456%209.13052%2012.4526%208.80078%2013.4315H8.00781L7.48145%2013.8846V14.1014L8.58398%2014.2147C8.53246%2014.4104%208.50162%2014.617%208.4707%2014.8231L7.47168%2014.7195V15.2654H6.77051L6.94629%2016.8943L8.4502%2017.0584C8.47077%2017.1716%208.48141%2017.2749%208.50195%2017.3777L6.9873%2017.2137L7.13184%2018.5526H7.67773L8.59473%2017.8836C8.63589%2018.038%208.67759%2018.1922%208.71875%2018.3465H8.65625V18.2234L7.67773%2018.9236V19.8309H7.5332L8.1416%2020.8611L9.53223%2020.4383V20.2684C9.68855%2020.5194%209.85276%2020.7622%2010.0273%2020.9949H9.7793L9.12012%2021.8397V22.5506H8.35742V21.7889H8.40918L8.01758%2021.119L7.24512%2021.3553V21.7576H6.48242V20.9949H6.76074L6.85352%2019.8406H6.3584V19.6658L5.50293%2020.1395V20.5623H4.74121V19.7996H4.93652L4.85449%2019.0067L4.07129%2018.9031V19.1199H3.30859V18.367H3.75195L4.2666%2017.4295H4.08105V16.9656L2.90625%2017.0789V17.3992H2.14453V16.6463H2.57715L2.89648%2015.9051L2.14453%2015.4822V15.493H1.38184V14.7401H2.07227L3.39062%2013.8436V13.535L1.7627%2013.2459V13.4627H1V12.7108H1.61816L2.28809%2012.0711L1.62891%2011.4637H1V10.701H1.7627V10.8865L3.38086%2010.577V10.2479L2.13379%209.42365V9.44415H1.37109V8.68146H2.13379V8.71271L2.90625%208.25861L2.58691%207.5379H2.14453V6.77521H2.90625V7.07404L4.05078%207.16681V6.66193H4.20508L3.73145%205.81622H3.28809V5.0545H4.05078V5.31134L4.84375%205.16779L4.94727%204.35333H4.7207V3.59064H5.48242V4.0545L6.33789%204.46661V4.24005H6.83301L6.76074%203.17853H6.46191V2.41583H7.22461V2.81818L7.97656%203.02423L8.37793%202.36505H8.35742V1.60236H9.12012V2.29279L9.74902%203.08575H10.2021L10.5215%201.96271H10.377V1.20001H11.1396V1.84943ZM16.6123%209.61896C20.1156%209.61919%2023.0009%2012.5043%2023.001%2016.0076C23.001%2019.511%2020.1156%2022.3961%2016.6123%2022.3963C13.1088%2022.3963%2010.2236%2019.5111%2010.2236%2016.0076C10.2237%2012.5042%2013.1088%209.61896%2016.6123%209.61896ZM8.25488%2021.0467L8.6875%2021.7781H8.93457V21.7889L9.55273%2020.9949H9.54297V20.6551L8.25488%2021.0467ZM6.99707%2020.9949H7.27539V21.1287L7.28613%2021.1395L7.94531%2020.9334L7.2959%2019.8406H7.08984L6.99707%2020.9949ZM16.6123%2011.0613C13.8302%2011.0613%2011.6661%2013.2255%2011.666%2016.0076C11.666%2018.7898%2013.8301%2020.9539%2016.6123%2020.9539C19.2913%2020.9537%2021.5576%2018.7897%2021.5576%2016.0076C21.5576%2013.2256%2019.3943%2011.0616%2016.6123%2011.0613ZM9.53223%2020.2684C9.52255%2020.2528%209.51156%2020.2381%209.50195%2020.2225L9.53223%2020.2117V20.2684ZM5.17383%2019.8104H5.52441V19.9031L6.37891%2019.4393V19.2225L5.09082%2019.0477L5.17383%2019.8104ZM18.1572%2013.4315C18.4662%2013.1224%2018.8784%2013.1227%2019.1875%2013.4315C19.4966%2013.6375%2019.4966%2014.1526%2019.1875%2014.4617L17.5391%2016.1111L19.1875%2017.7596C19.4966%2018.0687%2019.4966%2018.4807%2019.1875%2018.7899C18.8784%2019.0988%2018.4663%2019.0989%2018.1572%2018.7899L16.5088%2017.1414L14.8604%2018.7899C14.5513%2019.0989%2014.1392%2019.0988%2013.8301%2018.7899C13.5209%2018.4807%2013.5209%2018.0687%2013.8301%2017.7596L15.4785%2016.1111L13.8301%2014.4617C13.5209%2014.1526%2013.5209%2013.7406%2013.8301%2013.4315C14.1392%2013.1226%2014.5513%2013.1224%2014.8604%2013.4315L16.5088%2015.0799L18.1572%2013.4315ZM5.38965%2017.4402H4.91602L5.07031%2018.8309L6.37891%2019.0067V18.5526H6.8125L6.65723%2017.1717L5.38965%2017.0379V17.4402ZM3.99902%2018.367H4.0918V18.6883L4.84375%2018.7899L4.69922%2017.4295H4.51465L3.99902%2018.367ZM2.81348%2016.6463H2.92773V16.8729L4.0918%2016.7596V16.5643L3.0918%2016.0076L2.81348%2016.6463ZM5.38965%2016.1619V16.7186L6.62695%2016.8524H6.63672L6.46191%2015.2557H6.0293L5.38965%2016.1619ZM4.69922%2014.1834H3.875L3.1748%2015.8123L4.0918%2016.327V16.1317H5.0293L5.64746%2015.2557H5.55469V14.1736L4.69922%2013.9061V14.1834ZM2.16504%2014.9363V15.2449L2.99902%2015.7088L3.65918%2014.1834H3.42188V14.0906L2.16504%2014.9363ZM4.66895%2012.8748H4.70996V13.576L5.56543%2013.8436V13.3494H5.69922V13.3387L5.04004%2012.3192L4.66895%2012.8748ZM5.21484%2012.0506L6.05957%2013.3592H7.11133L7.69824%2012.8543V11.3192L7.11133%2010.7938H6.0498L5.21484%2012.0506ZM1.7832%2012.8445V13.0301L1.79395%2013.0399L3.41113%2013.3289V13.0916L2.46387%2012.2059L1.7832%2012.8445ZM2.60742%2012.0613H2.59766L3.49414%2012.8856H4.28711L4.84375%2012.0506L4.28711%2011.1854H3.52539L2.60742%2012.0613ZM1.7832%2011.0926V11.2986L2.46387%2011.9276L3.40137%2011.0311V10.783L1.7832%2011.0926ZM4.68945%2010.5262V11.1951H4.66895L5.04004%2011.7723L5.68848%2010.7938L5.67871%2010.8045H5.54492V10.2889L4.68945%2010.5262ZM7.45117%2010.0516V10.2781L7.97656%2010.742H8.8623L8.94531%209.88654L7.45117%2010.0516ZM3.21582%208.32111L3.90625%209.89728H4.70996V10.1961L5.56543%209.9588V8.88751H5.64746L4.98828%207.93927L5.00879%207.9295H4.10156V7.79474L3.21582%208.32111ZM2.16504%208.92853V9.15509L2.1543%209.16583L3.40137%209.99005V9.89728H3.65918L3.02051%208.42365L2.16504%208.92853ZM6.93555%207.18732H6.92578L6.75%208.89728H7.46094V9.43341L9.01758%209.25861L9.17188%207.64044H8.64648V6.98126L6.93555%207.18732ZM9.65625%209.18634L10.4902%209.09357V8.20685L9.99609%207.64044H9.80078L9.65625%209.18634ZM5.36914%207.37286V7.93927H5.35938L6.01855%208.88751H6.44141L6.61621%207.21857L5.36914%207.37286ZM2.9375%207.5174H2.84961L3.1123%208.12482L4.08105%207.55841V7.35236L2.9375%207.25958V7.5174ZM19.5693%207.33185V7.5379L20.5791%208.11505L20.8359%207.50665H20.7432V7.23907L19.5693%207.33185ZM10.5527%206.1463V7.34161L11.0371%207.90802H12.582L13.1084%207.31036H13.1182V6.17755L11.8203%205.18829L10.5527%206.1463ZM17.292%205.5174L17.2812%205.52814H16.8486L17.0029%206.909L18.2705%207.06329V6.64142H18.7441L18.6006%205.28107L17.292%205.08478V5.5174ZM5.08105%205.33282L4.91602%206.64142H5.37988V7.05353L6.65723%206.89825L6.80176%205.52814H6.37891V5.10529L5.08105%205.33282ZM15.0244%206.20782V6.66193L16.6943%206.86798L16.5391%205.52814H15.9932L15.0244%206.20782ZM6.97656%206.85724L8.65625%206.65118V6.22931L7.66699%205.52814H7.12012L6.97656%206.85724ZM4.0918%205.5174V5.80646H4.00879L4.4834%206.65118H4.70996L4.86426%205.38361L4.0918%205.5174ZM18.9502%206.63068H19.167L19.6406%205.79572H19.6104V5.4256L19.5996%205.41486L18.8066%205.30157L18.9502%206.63068ZM8.11035%203.2713L7.52344%204.25079H7.66699V5.15704L8.65625%205.85822V5.73419H9.61523L9.88281%204.37384H9.52246V3.65314L8.12109%203.26154L8.11035%203.2713ZM14.1279%203.6629V4.38458H13.7676L14.0352%205.76544H15.0352V5.82697L16.0039%205.14728V4.24005H16.127L15.5498%203.2713L14.1279%203.6629ZM13.4482%204.37384H12.8916L12.0771%204.992L13.1182%205.78595V5.74493H13.7158L13.4482%204.36407V4.37384ZM9.92383%205.72443L10.5625%205.74493L11.5625%204.98224L10.7686%204.37384H10.1914V4.36407L9.92383%205.72443ZM5.51367%204.34357H5.19434L5.18359%204.35333L5.09082%205.12677L6.36914%204.89923V4.69318L5.51367%204.28107V4.34357ZM17.292%204.65216V4.86896L18.5801%205.0545L18.5078%204.33282H18.1885V4.18829L17.292%204.65216ZM10.8203%203.3338V4.02325L11.8301%204.79669L12.8301%204.034V3.36407L11.7891%202.57111V2.56036L10.8203%203.3338ZM7.26562%203.17853H6.99707L7.06934%204.24005H7.27539L7.9043%203.19904L7.26562%203.02423V3.17853ZM15.7559%203.20978L16.3643%204.23029H16.5908L16.6836%203.18927H16.4365V3.02618L15.7559%203.20978ZM8.65625%202.37482L8.22363%203.08575L9.52246%203.44708V3.0965L8.94531%202.36505H8.65625V2.37482ZM14.1074%203.10626V3.44708L15.4268%203.08575L14.9941%202.35431H14.7158L14.1074%203.10626ZM10.46%203.0965H10.7998L11.624%202.43732L10.9961%201.96271H10.7793L10.46%203.0965ZM11.9541%202.42657L12.8301%203.0965V3.07599H13.1904L12.8711%201.96271H12.541V1.95294L11.9541%202.42657Z%22%20fill%3D%22currentColor%22%2F%3E%0A%3C%2Fsvg%3E%0A",
	)

	// Add block-specific parameter
	desc.Parameters = append(desc.Parameters,
		[]action_kit_api.ActionParameter{
			{
				Name:  "-response-header-",
				Type:  action_kit_api.ActionParameterTypeHeader,
				Label: "Response",
			},
			{
				Name:         "responseStatusCode",
				Label:        "Status Code",
				Description:  extutil.Ptr("The status code which should get returned."),
				Type:         action_kit_api.ActionParameterTypeInteger,
				MinValue:     extutil.Ptr(100),
				MaxValue:     extutil.Ptr(999),
				Required:     extutil.Ptr(true),
				DefaultValue: extutil.Ptr("503"),
			},
			{
				Name:  "-conditions-separator-",
				Label: "-",
				Type:  action_kit_api.ActionParameterTypeSeparator,
			},
			{
				Name:  "-conditions-header-",
				Type:  action_kit_api.ActionParameterTypeHeader,
				Label: "Conditions",
			},
			{
				Name:        "conditionPathPattern",
				Label:       "Path Pattern",
				Description: extutil.Ptr("The path patterns to compare against the request URL."),
				Type:        action_kit_api.ActionParameterTypeRegex,
				Required:    extutil.Ptr(false),
			},
			{
				Name:         "conditionHttpMethod",
				Label:        "HTTP Method",
				Description:  extutil.Ptr("The name of the request method."),
				Type:         action_kit_api.ActionParameterTypeString,
				DefaultValue: extutil.Ptr("GET"),
				Required:     extutil.Ptr(false),
				Options: extutil.Ptr([]action_kit_api.ParameterOption{
					action_kit_api.ExplicitParameterOption{
						Label: "*",
						Value: "*",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "GET",
						Value: "GET",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "POST",
						Value: "POST",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "PUT",
						Value: "PUT",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "PATCH",
						Value: "PATCH",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "HEAD",
						Value: "HEAD",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "DELETE",
						Value: "DELETE",
					},
				}),
			},
			{
				Name:        "conditionHttpHeader",
				Label:       "HTTP Header",
				Description: extutil.Ptr("The name of the HTTP header field with a maximum size of 40 characters. And a value to compare against the value of the HTTP header. The maximum size of each string is 128 characters. The comparison strings are case insensitive. The following wildcard characters are supported: * (matches 0 or more characters) and ? (matches exactly 1 character). Currently only a single header name with a single value is allowed."),
				Type:        action_kit_api.ActionParameterTypeKeyValue,
				Required:    extutil.Ptr(false),
			},
			//{
			//	Name:        "conditionDownstreamServiceName",
			//	Label:       "Downstream Service Name",
			//	Description: extutil.Ptr("The name of the downstream service to compare against the request URL. E.g. /card is the path to the card-service, but card-service in the name of the service."),
			//	Type:        action_kit_api.ActionParameterTypeRegex,
			//	Required:    extutil.Ptr(false),
			//},
		}...,
	)

	return desc
}

func (a *HAProxyBlockTrafficAction) Prepare(_ context.Context, state *HAProxyBlockTrafficState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	ingress, err := prepareHAProxyAction(&state.HAProxyBaseState, request)
	if err != nil {
		return nil, err
	}

	state.ResponseStatusCode = extutil.ToInt(request.Config["responseStatusCode"])

	state.ConditionPathPattern = extutil.ToString(request.Config["conditionPathPattern"])
	state.ConditionHttpMethod = extutil.ToString(request.Config["conditionHttpMethod"])
	if (request.Config["conditionHttpHeader"]) != nil {
		state.ConditionHttpHeader, err = extutil.ToKeyValue(request.Config, "conditionHttpHeader")
		if err != nil {
			return nil, err
		}
	}
	//state.ConditionDownstreamServiceName = extutil.ToString(request.Config["conditionDownstreamServiceName"])

	if state.ConditionPathPattern != "" {
		//Check if annotation for block already exists
		existingLines := strings.Split(ingress.Annotations[AnnotationKey], "\n")
		// Check if a rule with the same path already exists
		for _, line := range existingLines {
			if strings.HasPrefix(line, "http-request return status") && strings.Contains(line, fmt.Sprintf("if { path_reg %s }", state.ConditionPathPattern)) {
				return nil, fmt.Errorf("a rule for path %s already exists", state.ConditionPathPattern)
			}
		}
	}

	var configBuilder strings.Builder
	configBuilder.WriteString(getStartMarker(state.ExecutionId) + "\n")

	// Define ACLs for each condition
	aclIdPrefix := strings.Replace(state.ExecutionId.String()[0:8], "-", "_", -1)
	var aclDefinitions []string
	var aclRefs []string

	if state.ConditionHttpMethod != "" && state.ConditionHttpMethod != "*" {
		aclName := fmt.Sprintf("sb_method_%s", aclIdPrefix)
		aclDefinitions = append(aclDefinitions, fmt.Sprintf("acl %s method %s", aclName, state.ConditionHttpMethod))
		aclRefs = append(aclRefs, aclName)
	}

	if state.ConditionHttpHeader != nil {
		for headerName, headerValue := range state.ConditionHttpHeader {
			aclName := fmt.Sprintf("sb_hdr_%s_%s", strings.Replace(headerName, "-", "_", -1), aclIdPrefix)
			aclDefinitions = append(aclDefinitions, fmt.Sprintf("acl %s hdr(%s) -m reg %s", aclName, headerName, headerValue))
			aclRefs = append(aclRefs, aclName)
		}
	}

	if state.ConditionPathPattern != "" {
		aclName := fmt.Sprintf("sb_path_%s", aclIdPrefix)
		aclDefinitions = append(aclDefinitions, fmt.Sprintf("acl %s path_reg %s", aclName, state.ConditionPathPattern))
		aclRefs = append(aclRefs, aclName)
	}

	// Add all ACL definitions to config
	for _, aclDef := range aclDefinitions {
		configBuilder.WriteString(aclDef + "\n")
	}

	// Create the rule with the defined ACLs
	if len(aclRefs) > 0 {
		// Use AND logic between conditions (default behavior in HAProxy)
		combinedCondition := strings.Join(aclRefs, " ")
		configBuilder.WriteString(fmt.Sprintf("http-request return status %d if %s\n", state.ResponseStatusCode, combinedCondition))
	} else {
		return nil, fmt.Errorf("at least one condition is required")
	}

	configBuilder.WriteString(getEndMarker(state.ExecutionId) + "\n")
	state.AnnotationConfig = configBuilder.String()

	return nil, nil
}

func (a *HAProxyBlockTrafficAction) Start(ctx context.Context, state *HAProxyBlockTrafficState) (*action_kit_api.StartResult, error) {
	if err := startHAProxyAction(&state.HAProxyBaseState, state.AnnotationConfig); err != nil {
		return nil, err
	}

	return nil, nil
}

func (a *HAProxyBlockTrafficAction) Stop(_ context.Context, state *HAProxyBlockTrafficState) (*action_kit_api.StopResult, error) {
	if err := stopHAProxyAction(&state.HAProxyBaseState); err != nil {
		return nil, err
	}

	return nil, nil
}

//
//// Example function to add header blocking to an ingress
//func (c *Client) BlockRequestsByHeader(ctx context.Context, namespace, ingressName, headerName, headerValue string, executionId uuid.UUID) error {
//	// Create configuration block with markers for later removal
//	configSnippet := fmt.Sprintf(`# BEGIN STEADYBIT - %s
//   acl blocked_header hdr(%s) -i %s
//   http-request deny if blocked_header
//   # END STEADYBIT - %s`, executionId, headerName, headerValue, executionId)
//
//	// Add this configuration to the ingress
//	return c.UpdateIngressAnnotation(ctx, namespace, ingressName, "haproxy.org/configuration-snippet", configSnippet)
//}
