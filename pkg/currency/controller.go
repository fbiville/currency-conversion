package currency

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type currencyController struct {
	service Converter
}

func NewController(service Converter) http.Handler {
	return &currencyController{service: service}
}

type conversionPayload struct {
	SourceCurrency Currency    `json:"sourceCurrency"`
	SourceValue    json.Number `json:"sourceValue"`
	TargetCurrency Currency    `json:"targetCurrency"`
}

func (controller *currencyController) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	response.Header().Add("Content-Type", "application/json")
	if request.Method != "POST" {
		response.WriteHeader(405)
		response.Write([]byte(`{"error": "only POST requests are supported")`))
		return
	}
	if ok := validateHeader(request.Header, "Content-Type", "application/json"); !ok {
		response.WriteHeader(415)
		response.Write([]byte(`{"error": "only JSON requests are supported")`))
		return
	}
	if ok := validateHeader(request.Header, "Accept", "application/json"); !ok {
		response.WriteHeader(406)
		response.Write([]byte(`{"error": "only JSON responses are supported")`))
		return
	}

	payload := conversionPayload{}
	jsonDecoder := newJsonDecoder(request.Body)
	if err := jsonDecoder.Decode(&payload); err != nil {
		sendBadRequest(response, err.Error())
		return
	}
	result, err := controller.service.Convert(
		Amount{
			Quantity: payload.SourceValue,
			Currency: payload.SourceCurrency,
		},
		payload.TargetCurrency)
	if err != nil {
		handleError(response, payload, err)
		return
	}
	response.WriteHeader(200)
	response.Write([]byte(fmt.Sprintf(`{"currency": "%s", "value": %s}`,
		result.Currency,
		result.Quantity)))
}

func handleError(response http.ResponseWriter, payload conversionPayload, err error) {
	sourceCurrency := payload.SourceCurrency
	targetCurrency := payload.TargetCurrency
	amount := payload.SourceValue
	switch err {
	case InvalidSourceCurrency:
		sendBadRequest(response, fmt.Sprintf("invalid source currency %s", sourceCurrency))
	case InvalidTargetCurrency:
		sendBadRequest(response, fmt.Sprintf("invalid target currency %s", targetCurrency))
	case InvalidConversionAmount:
		sendBadRequest(response, fmt.Sprintf("invalid conversion amount %s", amount))
	default:
		sendServerError(response, err.Error())
	}
}

func sendBadRequest(response http.ResponseWriter, msg string) {
	response.WriteHeader(400)
	response.Write([]byte(fmt.Sprintf(`{"error": "%s")`, msg)))
}

func sendServerError(response http.ResponseWriter, msg string) {
	response.WriteHeader(500)
	response.Write([]byte(fmt.Sprintf(`{"error": "%s")`, msg)))
}

func validateHeader(headers http.Header, header, expectedValue string) bool {
	values := headers.Values(header)
	if len(values) == 0 {
		return true
	}
	// YOLO content negotiation
	slashIndex := strings.Index(expectedValue, "/")
	if slashIndex == -1 {
		panic(fmt.Sprintf("invalid MIME type %s", expectedValue))
	}
	okValues := []string{
		expectedValue,
		fmt.Sprintf("%s/*", expectedValue[0:slashIndex]),
		"*/*",
	}
	for _, rawValue := range values {
		for _, value := range extractValues(rawValue) {
			for _, okValue := range okValues {
				if value == okValue {
					return true
				}
			}
		}
	}
	return false
}

func extractValues(rawValue string) []string {
	var result []string
	for _, value := range strings.Split(rawValue, ",") {
		value := strings.Trim(value, " ")
		if weightStart := strings.Index(value, ";"); weightStart > -1 {
			value = value[0:weightStart]
		}
		result = append(result, value)
	}
	return result
}
