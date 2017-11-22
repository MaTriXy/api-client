# api-client
**Make API requests as quickly as possible with ease**: api-client is a generic api client with built in rate limiting capabilities.  Use `api-client` as a base struct to quickly create JSON-consuming api-functionality that supports QPS rate limits. 

## Use Case
Suppose there's a JSON endpoint you want to consume, for example, some REST API. To achieve optimal network performance while taking advantage of go's concurrency capabilities, you would want to consume the API's response as fast as possible by making as many requests as fast as possible. 

This typically means something like a loop that keeps on firing requests to the API until you have all the data you need. The problem is that firing requests is a lot faster than receiving a response. This means that if you don't take any action to block the execution at a certain rate, you may end up firing requests too quickly to the target server, which will most probably fail (you will exceed either the OS's outbound http request limits or the servers capacity or quota, or both). 

For this reason, whenever you write a client api of this sort, you end up writing quite a lot of code to do the same - make the request, parse the response, block the execution to limit the rate to certain QPS, etc. `api-client` provides an easy interface to coping with this. 

## Example `api-client` Implementation

See WalkScore.com API client as an example implementation: 


Using the client is easy. Inject the API key and you are ready to go:

```go
	c, err := walkscore.NewClient(walkscore.WithAPIKey("wsapikey", "123"))
	if err != nil {
		log.Fatal(err)
	}

	tonsOfRequestsToMake := []*walkscore.Request{
		&Request{
			Address:   "2025 1st Avenue Suite 500, Seattle, WA 98121",
			Latitude:  "47.6114338",
			Longitude: "-122.3460171",
		},
		...
		...
		...
	}
```

Now you can call `c.GetJSON` **as quickly as possible**, without having to worry about overloading your network or the walkscore server - the client will internally manage the request queue to support a default QPS rate limit of 10. You can change the default rate limit by using the `WithRateLimit` client option. 	

```go
	// go over all requests, send them to the client as fast as possible using an anonymouse function - it will take care
	// of adjusting the rate limits
	for _, r := range tonsOfRequestsToMake {
		go func() {
			var resp *Response
			err = c.GetJSON(context.Background(), r, &resp) 
			if err != nil {
				log.Fatal(err)
			}	
			// do something with the response
		}()
	}
```

