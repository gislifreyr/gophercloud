package gophercloud

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/rackspace/gophercloud/testhelper"
)

func createClient() *ServiceClient {
	return &ServiceClient{
		Provider: &ProviderClient{TokenID: "abc123"},
		Endpoint: testhelper.Endpoint(),
	}
}

// SinglePage sample and test cases.

func ExtractSingleInts(page Page) ([]int, error) {
	var response struct {
		Ints []int `mapstructure:"ints"`
	}

	err := mapstructure.Decode(page.(SinglePage).Body, &response)
	if err != nil {
		return nil, err
	}

	return response.Ints, nil
}

func setupSinglePaged() Pager {
	testhelper.SetupHTTP()
	client := createClient()

	testhelper.Mux.HandleFunc("/only", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{ "ints": [1, 2, 3] }`)
	})

	return NewSinglePager(client, testhelper.Server.URL+"/only")
}

func TestEnumerateSinglePaged(t *testing.T) {
	callCount := 0
	pager := setupSinglePaged()
	defer testhelper.TeardownHTTP()

	err := pager.EachPage(func(page Page) (bool, error) {
		callCount++

		expected := []int{1, 2, 3}
		actual, err := ExtractSingleInts(page)
		if err != nil {
			return false, err
		}
		if !reflect.DeepEqual(expected, actual) {
			t.Errorf("Expected %v, but was %v", expected, actual)
		}
		return true, nil
	})
	if err != nil {
		t.Fatalf("Unexpected error calling EachPage: %v", err)
	}

	if callCount != 1 {
		t.Errorf("Callback was invoked %d times", callCount)
	}
}

// LinkedPager sample and test cases.

func ExtractLinkedInts(page Page) ([]int, error) {
	var response struct {
		Ints []int `mapstructure:"ints"`
	}

	err := mapstructure.Decode(page.(LinkedPage).Body, &response)
	if err != nil {
		return nil, err
	}

	return response.Ints, nil
}

func createLinked(t *testing.T) Pager {
	testhelper.SetupHTTP()

	testhelper.Mux.HandleFunc("/page1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{ "ints": [1, 2, 3], "links": { "next": "%s/page2" } }`, testhelper.Server.URL)
	})

	testhelper.Mux.HandleFunc("/page2", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{ "ints": [4, 5, 6], "links": { "next": "%s/page3" } }`, testhelper.Server.URL)
	})

	testhelper.Mux.HandleFunc("/page3", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{ "ints": [7, 8, 9], "links": { "next": null } }`)
	})

	client := createClient()

	return NewLinkedPager(client, testhelper.Server.URL+"/page1")
}

func TestEnumerateLinked(t *testing.T) {
	pager := createLinked(t)
	defer testhelper.TeardownHTTP()

	callCount := 0
	err := pager.EachPage(func(page Page) (bool, error) {
		actual, err := ExtractLinkedInts(page)
		if err != nil {
			return false, err
		}

		t.Logf("Handler invoked with %v", actual)

		var expected []int
		switch callCount {
		case 0:
			expected = []int{1, 2, 3}
		case 1:
			expected = []int{4, 5, 6}
		case 2:
			expected = []int{7, 8, 9}
		default:
			t.Fatalf("Unexpected call count: %d", callCount)
			return false, nil
		}

		if !reflect.DeepEqual(expected, actual) {
			t.Errorf("Call %d: Expected %#v, but was %#v", callCount, expected, actual)
		}

		callCount++
		return true, nil
	})
	if err != nil {
		t.Errorf("Unexpected error for page iteration: %v", err)
	}

	if callCount != 3 {
		t.Errorf("Expected 3 calls, but was %d", callCount)
	}
}
