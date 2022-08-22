package currency_test

import (
	"errors"
	"fmt"
	"github.com/fbiville/currency-conversion/pkg/currency"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCurrencyConversion(outer *testing.T) {

	outer.Run("converts currency", func(t *testing.T) {
		server := httptest.NewServer(jsonHandler(`{
	"success": true,
	"query": {"from": "EUR", "to": "USD", "amount": 10},
	"info": {"timestamp": 1661180044, "rate": 0.95},
	"date": "2022-08-22",
	"result": 9.5}`))
		defer server.Close()
		amount := currency.Amount{Currency: "EUR", Quantity: "10"}
		converter := currency.NewConverter(server.URL, "alan-key")

		result, err := converter.Convert(amount, "USD")

		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
		if result.Quantity != "9.5" || result.Currency != "USD" {
			t.Errorf("expected 9.5 USD, got %v", result)
		}
	})

	type failure struct {
		reason  string
		handler http.HandlerFunc
		err     error
	}

	for _, failure := range []failure{
		{reason: "source currency",
			handler: badRequest("invalid_from_currency"),
			err:     currency.InvalidSourceCurrency},
		{reason: "target currency",
			handler: badRequest("invalid_to_currency"),
			err:     currency.InvalidTargetCurrency},
		{reason: "conversion amount",
			handler: badRequest("invalid_conversion_amount"),
			err:     currency.InvalidConversionAmount},
		{reason: "whatever else",
			handler: badRequest("something_else"),
			err:     errors.New(`error "something_else": oopsie`)},
	} {
		outer.Run(fmt.Sprintf("bad request because of %s", failure.reason),
			func(t *testing.T) {
				server := httptest.NewServer(failure.handler)
				defer server.Close()
				converter := currency.NewConverter(server.URL, "alan-key")
				amount := currency.Amount{Currency: "irrelevant", Quantity: "irrelevant"}

				_, err := converter.Convert(amount, "irrelevant")

				if err.Error() != failure.err.Error() {
					t.Errorf("expected error %q but got %q", failure.err, err)
				}
			})
	}

	outer.Run("fails because of upstream's unexpected HTTP response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			response.WriteHeader(500)
			response.Header().Add("Content-Type", "text/plain")
			_, _ = response.Write([]byte("nope"))
		}))
		defer server.Close()
		converter := currency.NewConverter(server.URL, "alan-key")
		amount := currency.Amount{Currency: "irrelevant", Quantity: "irrelevant"}
		expectedErr := errors.New("unexpected error (upstream status 500): nope")

		_, err := converter.Convert(amount, "irrelevant")

		if err.Error() != expectedErr.Error() {
			t.Errorf("expected error %q but got %q", expectedErr, err)
		}
	})

}

func jsonHandler(json string) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(200)
		response.Header().Add("Content-Type", "application/json")
		_, _ = response.Write([]byte(json))
	}
}

func badRequest(errorCode string) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(400)
		response.Header().Add("Content-Type", "application/json")
		errorResponseBody := fmt.Sprintf(
			`{"error": {"code": "%s", "message": "oopsie"}}`,
			errorCode)
		_, _ = response.Write([]byte(errorResponseBody))
	}
}
