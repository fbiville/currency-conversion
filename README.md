# Currency converter API

## Prereqs
Install the latest [Go version](https://go.dev/).


## Run

### Start Server

```go
go run ./cmd/server
```

### Run Client

Send a currency conversion request:

```shell
curl  --header 'Content-Type: application/json' \ 
      --data '{"sourceCurrency": "EUR", "sourceValue": 7000, "targetCurrency": "TRY"}' \
       'http://[::]:65012'
```

Note: replace the address by the actual address printed by the first command.

