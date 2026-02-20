
package main

import (
	"github.com/itchyny/gojq"
	"encoding/json"
	"fmt"
	"net/http"
	"io"
)

func jqProcessUrl(url string, query string) []string {
	jq, err := gojq.Parse(query)
	ohno(err)

	res, err := http.Get(url)
	ohno(err)

	body, err := io.ReadAll(res.Body)
	ohno(err)
	res.Body.Close()
	if res.StatusCode > 300 {
		ohno(fmt.Errorf("Unexpected status code: %v", res.StatusCode))
	}

	var decoded_json map[string]any
	err = json.Unmarshal(body, &decoded_json)
	ohno(err)

	iter := jq.Run(decoded_json)
	results := []string{}
	for {
		v, hasmore := iter.Next()
		if !hasmore {
			break
		}
		if err, ok := v.(error); ok {
			if err, ok := err.(*gojq.HaltError); ok && err.Value() == nil {
				break
			}
			ohno(err)
		}
		s, ok := v.(string)
		if !ok {
			ohno(fmt.Errorf("Unexpected type %v", v))
		}
		fmt.Println(s)
		results = append(results, s)
	}
	return results
}
