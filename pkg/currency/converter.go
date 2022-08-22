package currency

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

var zeroAmount = Amount{}

type Currency string

type Amount struct {
	Quantity json.Number
	Currency Currency
}

type Converter interface {
	Convert(amount Amount, targetCurrency Currency) (Amount, error)
}

type apiLayerConverter struct {
	key    string
	uri    string
	client *http.Client
}

func NewConverter(baseUri, apiKey string) Converter {
	return &apiLayerConverter{
		key:    apiKey,
		uri:    fmt.Sprintf("%s/exchangerates_data/convert", baseUri),
		client: &http.Client{},
	}
}

func (converter *apiLayerConverter) Convert(amount Amount, targetCurrency Currency) (Amount, error) {
	request, err := converter.newConversionRequest(amount, targetCurrency)
	if err != nil {
		return zeroAmount, err
	}

	response, err := converter.client.Do(request)
	if err != nil {
		return zeroAmount, fmt.Errorf("failed to perform conversion: %w", err)
	}
	statusCode := response.StatusCode
	if statusCode == http.StatusBadRequest {
		return zeroAmount, converter.extractBadRequestError(response)
	}
	if statusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return zeroAmount, fmt.Errorf("unexpected error (upstream status %d): %s",
			statusCode,
			body)
	}
	var result map[string]any
	jsonDecoder := newJsonDecoder(response.Body)
	if err = jsonDecoder.Decode(&result); err != nil {
		return zeroAmount, fmt.Errorf("could not read conversion response: %w", err)
	}
	rawAmount := result["result"]
	return Amount{
		Quantity: rawAmount.(json.Number),
		Currency: targetCurrency,
	}, nil
}

func (converter *apiLayerConverter) newConversionRequest(amount Amount, targetCurrency Currency) (*http.Request, error) {
	request, err := http.NewRequest("GET", converter.uri, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create conversion request: %w", err)
	}
	headers := request.Header
	headers.Set("Accept", "application/json")
	headers.Set("apikey", converter.key)
	queryString := url.Values{}
	queryString.Add("from", string(amount.Currency))
	queryString.Add("amount", string(amount.Quantity))
	queryString.Add("to", string(targetCurrency))
	request.URL.RawQuery = queryString.Encode()
	return request, nil
}

func (converter *apiLayerConverter) extractBadRequestError(response *http.Response) error {
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read conversion response: %w", err)
	}
	errorResponse := apiLayerErrorResponse{}
	if err = json.Unmarshal(responseBody, &errorResponse); err != nil {
		return fmt.Errorf("failed to parse conversion error response: %w", err)
	}
	errorCode := errorResponse.Error.Code
	switch errorCode {
	case "invalid_from_currency":
		return InvalidSourceCurrency
	case "invalid_to_currency":
		return InvalidTargetCurrency
	case "invalid_conversion_amount":
		return InvalidConversionAmount
	default:
		return fmt.Errorf("error %q: %v", errorCode, errorResponse.Error.Message)
	}
}

type apiLayerErrorResponse struct {
	Error apiLayerError `json:"error"`
}

type apiLayerError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func newJsonDecoder(reader io.Reader) *json.Decoder {
	jsonDecoder := json.NewDecoder(reader)
	jsonDecoder.UseNumber()
	return jsonDecoder
}

type ConversionError string

func (c ConversionError) Error() string {
	return string(c)
}

const InvalidSourceCurrency = ConversionError("invalid source currency")
const InvalidTargetCurrency = ConversionError("invalid target currency")
const InvalidConversionAmount = ConversionError("invalid conversion amount")
