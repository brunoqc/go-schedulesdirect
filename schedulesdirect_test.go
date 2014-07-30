package schedulesdirect

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// inspired by https://willnorris.com/2013/08/testing-in-go-github
var (
	mux    *http.ServeMux
	server *httptest.Server
	client sdclient
)

func setup() {
	// test server
	mux = http.NewServeMux()
	server = httptest.NewServer(mux)

	// schedules direct client configured to use test server
	client = sdclient{
		baseURL: server.URL,
	}
}

func testMethod(t *testing.T, r *http.Request, expectedMethod string) {
	if r.Method != expectedMethod {
		t.Fatalf("method (%s) != expectedMethod (%s)", r.Method, expectedMethod)
	}
}

func testHeader(t *testing.T, r *http.Request, header, expectedValue string) {
	if r.Header.Get(header) != expectedValue {
		t.Fatalf("token (%s) != expectedValue (%s)", r.Header.Get("token"), expectedValue)
	}
}

func testUrlParameter(t *testing.T, r *http.Request, parameter, expectedValue string) {
	if parameter == "postalcode" {
		// There's a bug with postal code containing a space
		// https://github.com/SchedulesDirect/JSON-Service/issues/31
		expectedValue = strings.Replace(expectedValue, " ", "", -1)
	}

	p := r.URL.Query().Get(parameter)

	if p != expectedValue {
		t.Fatalf("parameter (%s (%s)) != expectedValue (%s)", parameter, p, expectedValue)
	}
}

func TestGetTokenOK(t *testing.T) {
	setup()

	mux.HandleFunc(apiVersion+"/token",
		func(w http.ResponseWriter, r *http.Request) {
			testMethod(t, r, "POST")

			var tokenReq tokenRequest

			errDecode := json.NewDecoder(r.Body).Decode(&tokenReq)
			if errDecode != nil {
				t.Fatal(errDecode)
			}

			fmt.Fprint(w, `{"code":0,"message":"OK","serverID":"serverID1","token":"token1"}`)
		},
	)

	token, errToken := client.GetToken("user1", "pass1")
	if errToken != nil {
		t.Fatal(errToken)
	}

	if token != "token1" {
		t.Fatalf("token doesn't match")
	}
}

func TestNewClient(t *testing.T) {
	client := NewClient()
	if client.baseURL != baseurl {
		t.Fail()
	}
}

func TestHashPassword(t *testing.T) {
	if hashPassword("testpassword") != "8bb6118f8fd6935ad0876a3be34a717d32708ffd" {
		t.Fail()
	}

}

func TestGetTokenInvalidUser(t *testing.T) {
	setup()

	mux.HandleFunc(apiVersion+"/token",
		func(w http.ResponseWriter, r *http.Request) {
			testMethod(t, r, "POST")

			var tokenReq tokenRequest

			errDecode := json.NewDecoder(r.Body).Decode(&tokenReq)
			if errDecode != nil {
				t.Fatal(errDecode)
			}

			fmt.Fprint(w, `{"response":"INVALID_USER","code":4003,"serverID":"serverID1","message":"Invalid user.","datetime":"2014-07-29T01:00:28Z"}`)
		},
	)

	_, errToken := client.GetToken("user1", "pass1")
	if errToken != err_INVALID_USER {
		t.Fatalf("errToken != err_INVALID_USER (%s)", errToken.Error())
	}
}

func TestGetStatusOK(t *testing.T) {
	setup()

	mux.HandleFunc(apiVersion+"/status",
		func(w http.ResponseWriter, r *http.Request) {
			testMethod(t, r, "GET")
			testHeader(t, r, "token", "token1")

			fmt.Fprint(w, `{"account":{"expires":"2014-09-26T19:07:28Z","messages":[],"maxLineups":4,"nextSuggestedConnectTime":"2014-07-29T22:43:22Z"},"lineups":[],"lastDataUpdate":"2014-07-28T14:48:59Z","notifications":[],"systemStatus":[{"date":"2012-12-17T16:24:47Z","status":"Online","details":"All servers running normally."}],"serverID":"serverID1","code":0}`)
		},
	)

	_, err := client.GetStatus("token1")
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetHeadendsOK(t *testing.T) {
	setup()

	mux.HandleFunc(apiVersion+"/headends",
		func(w http.ResponseWriter, r *http.Request) {
			testMethod(t, r, "GET")
			testHeader(t, r, "token", "token1")
			testUrlParameter(t, r, "country", "CAN")
			testUrlParameter(t, r, "postalcode", "H0H 0H0")

			fmt.Fprint(w, `{"0000001":{"lineups":[{"name":"name1","uri":"uri1"},{"name":"name2","uri":"uri2"}],"location":"City1","type":"type1"},"0000002":{"lineups":[{"name":"name3","uri":"uri3"}],"location":"City2","type":"type2"}}`)
		},
	)

	headends, errGetHeadends := client.GetHeadends("token1", "CAN", "H0H 0H0")
	if errGetHeadends != nil {
		t.Fatal(errGetHeadends)
	}

	if len(headends) != 2 {
		t.Fatalf("len(headends) != 2: %d", len(headends))
	} else {
		if len(headends["0000001"].Lineups) != 2 {
			t.Fatalf(`len(headends["0000001"].Lineups) != 2: %d`, len(headends["0000001"].Lineups))
		} else if headends["0000001"].Lineups[0].Name != "name1" {
			t.Fatalf(`headends["0000001"].Lineups[0].Name != "name1": %s`, headends["0000001"].Lineups[0].Name)
		}
		if len(headends["0000002"].Lineups) != 1 {
			t.Fatalf(`len(headends["0000002"].Lineups) != 1: %d`, len(headends["0000002"].Lineups))
		}
	}
}

func TestGetHeadendsFailsWithMessage(t *testing.T) {
	setup()

	mux.HandleFunc(apiVersion+"/headends",
		func(w http.ResponseWriter, r *http.Request) {
			testMethod(t, r, "GET")
			testHeader(t, r, "token", "token1")
			testUrlParameter(t, r, "country", "CAN")
			testUrlParameter(t, r, "postalcode", "H0H 0H0")

			fmt.Fprint(w, `{"response":"INVALID_PARAMETER:COUNTRY","code":2050,"serverID":"serverID1","message":"The COUNTRY parameter must be ISO-3166-1 alpha 3. See http:\/\/en.wikipedia.org\/wiki\/ISO_3166-1_alpha-3","datetime":"2014-07-29T23:16:52Z"}`)
		},
	)

	_, errGetHeadends := client.GetHeadends("token1", "CAN", "H0H 0H0")
	if errGetHeadends.Error() != "The COUNTRY parameter must be ISO-3166-1 alpha 3. See http://en.wikipedia.org/wiki/ISO_3166-1_alpha-3" {
		t.Fail()
	}
}

func TestGetHeadendsFailsWithMessage2(t *testing.T) {
	setup()

	mux.HandleFunc(apiVersion+"/headends",
		func(w http.ResponseWriter, r *http.Request) {
			testMethod(t, r, "GET")
			testHeader(t, r, "token", "token1")
			testUrlParameter(t, r, "country", "CAN")
			testUrlParameter(t, r, "postalcode", "H0H 0H0")

			fmt.Fprint(w, `{"response":"REQUIRED_PARAMETER_MISSING:COUNTRY","code":2004,"serverID":"serverID1","message":"In order to search for lineups, you must supply a 3-letter country parameter.","datetime":"2014-07-29T23:15:18Z"}`)
		},
	)

	_, errGetHeadends := client.GetHeadends("token1", "CAN", "H0H 0H0")
	if errGetHeadends.Error() != "In order to search for lineups, you must supply a 3-letter country parameter." {
		t.Fail()
	}
}

func TestAddLineupOK(t *testing.T) {
	setup()

	mux.HandleFunc("/20131021/lineups/CAN-0000001-X",
		func(w http.ResponseWriter, r *http.Request) {
			testMethod(t, r, "PUT")
			testHeader(t, r, "token", "token1")
			fmt.Fprint(w, `{"response":"OK","code":0,"serverID":"serverID1","message":"Added lineup.","changesRemaining":5,"datetime":"2014-07-30T01:50:59Z"}`)
		},
	)

	changesRemaining, errAddLineup := client.AddLineup("token1", "/20131021/lineups/CAN-0000001-X")
	if errAddLineup != nil {
		t.Fatal(errAddLineup)
	}

	if changesRemaining != 5 {
		t.Fail()
	}
}

func TestAddLineupFailsDuplicate(t *testing.T) {
	setup()

	mux.HandleFunc("/20131021/lineups/CAN-0000001-X",
		func(w http.ResponseWriter, r *http.Request) {
			testMethod(t, r, "PUT")
			testHeader(t, r, "token", "token1")
			fmt.Fprint(w, `{"response":"DUPLICATE_HEADEND","code":2100,"serverID":"serverID1","message":"Headend already in account.","datetime":"2014-07-30T02:01:37Z"}`)
		},
	)

	_, errAddLineup := client.AddLineup("token1", "/20131021/lineups/CAN-0000001-X")
	if errAddLineup == nil {
		t.Fail()
	} else if errAddLineup.Error() != "Headend already in account." {
		t.Fail()
	}
}

func TestAddLineupFailsInvalidLineup(t *testing.T) {
	setup()

	mux.HandleFunc("/20131021/lineups/CAN-0000001-X",
		func(w http.ResponseWriter, r *http.Request) {
			testMethod(t, r, "PUT")
			testHeader(t, r, "token", "token1")

			fmt.Fprint(w, `{"response":"INVALID_LINEUP","code":2105,"serverID":"serverID1","message":"The lineup you submitted doesn't exist.","datetime":"2014-07-30T02:02:04Z"}`)
		},
	)

	_, errAddLineup := client.AddLineup("token1", "/20131021/lineups/CAN-0000001-X")
	if errAddLineup == nil {
		t.Fail()
	} else if errAddLineup.Error() != "The lineup you submitted doesn't exist." {
		t.Fail()
	}
}

func TestAddLineupFailsInvalidUser(t *testing.T) {
	setup()

	mux.HandleFunc("/20131021/lineups/CAN-0000001-X",
		func(w http.ResponseWriter, r *http.Request) {
			testMethod(t, r, "PUT")
			testHeader(t, r, "token", "token1")
			fmt.Fprint(w, `{"response":"INVALID_USER","code":4003,"serverID":"serverID1","message":"Invalid user.","datetime":"2014-07-30T01:48:11Z"}`)
		},
	)

	_, errAddLineup := client.AddLineup("token1", "/20131021/lineups/CAN-0000001-X")
	if errAddLineup == nil {
		t.Fail()
	} else if errAddLineup.Error() != "Invalid user." {
		t.Fail()
	}
}

func TestDelLineupOK(t *testing.T) {
	setup()

	mux.HandleFunc("/20131021/lineups/CAN-0000001-X",
		func(w http.ResponseWriter, r *http.Request) {
			testMethod(t, r, "DELETE")
			testHeader(t, r, "token", "token1")
			fmt.Fprint(w, `{"response":"OK","code":0,"serverID":"serverid1","message":"Deleted lineup.","changesRemaining":"5","datetime":"2014-07-30T03:27:23Z"}`)
		},
	)

	changesRemaining, errDelLineup := client.DelLineup("token1", "/20131021/lineups/CAN-0000001-X")
	if errDelLineup != nil {
		t.Fatal(errDelLineup)
	}

	if changesRemaining != 5 {
		t.Fail()
	}
}

func TestDelLineupFailsInvalidLineup(t *testing.T) {
	setup()

	mux.HandleFunc("/20131021/lineups/CAN-0000001-X",
		func(w http.ResponseWriter, r *http.Request) {
			testMethod(t, r, "DELETE")
			testHeader(t, r, "token", "token1")

			fmt.Fprint(w, `{"response":"INVALID_LINEUP","code":2105,"serverID":"serverID1","message":"The lineup you submitted doesn't exist.","datetime":"2014-07-30T02:02:04Z"}`)
		},
	)

	_, errDelLineup := client.DelLineup("token1", "/20131021/lineups/CAN-0000001-X")
	if errDelLineup == nil {
		t.Fail()
	} else if errDelLineup.Error() != "The lineup you submitted doesn't exist." {
		t.Fail()
	}
}

func TestGetLineupsOK(t *testing.T) {
	setup()

	mux.HandleFunc(apiVersion+"/lineups",
		func(w http.ResponseWriter, r *http.Request) {
			testMethod(t, r, "GET")
			testHeader(t, r, "token", "token1")

			fmt.Fprint(w, `{"serverID":"serverid1","datetime":"2014-07-30T02:34:37Z","lineups":[{"name":"name1","type":"type1","location":"location1","uri":"uri1"}]}`)
		},
	)

	lineups, errGetLineups := client.GetLineups("token1")
	if errGetLineups != nil {
		t.Fatal(errGetLineups)
	}

	if len(lineups.Lineups) != 1 {
		t.Fatalf("len(lineups.Lineups) != 1: %d", len(lineups.Lineups))
	} else if lineups.Lineups[0].Name != "name1" {
		t.Fatal(`lineups.Lineups[0].Name != "name1": %s`, lineups.Lineups[0].Name)
	}
}

func TestGetLineupsFailsNoHeadends(t *testing.T) {
	setup()

	mux.HandleFunc(apiVersion+"/lineups",
		func(w http.ResponseWriter, r *http.Request) {
			testMethod(t, r, "GET")
			testHeader(t, r, "token", "token1")

			// bug with the web service?
			http.Error(w, "", http.StatusBadRequest)

			fmt.Fprint(w, `{"response":"NO_LINEUPS","code":4102,"serverID":"serverID1","message":"No lineups have been added to this account.","datetime":"2014-07-30T01:21:56Z"}`)
		},
	)

	_, errGetLineups := client.GetLineups("token1")
	if errGetLineups == nil {
		t.Fatal("errGetLineups == nil")
	} else if errGetLineups.Error() != "No lineups have been added to this account." {
		t.Fail()
	}
}

func TestGetChannelMappingOK(t *testing.T) {
	setup()

	mux.HandleFunc("/20131021/lineups/CAN-0000001-X",
		func(w http.ResponseWriter, r *http.Request) {
			testMethod(t, r, "GET")
			testHeader(t, r, "token", "token1")
			fmt.Fprint(w, `{"map": [{"channel": "101","stationID": "10001"},{"channel": "1933","stationID": "10001"}],"metadata": {"lineup": "CAN-0000000-X","modified": "2014-07-29T16:38:09Z","transport": "transport1"},"stations": [{"affiliate": "affiliate1","broadcaster": {"city": "Unknown","country": "Unknown","postalcode": "00000"},"callsign": "callsign1","language": "en","name": "name1","stationID": "10001"},       {"callsign": "callsign2","language": "en","logo": {"URL": "https://domain/path/file.png","dimension": "w=360px|h=270px","md5": "ba5b5b5085baac6da247564039c03c9e"},"name": "name2","stationID": "10002"}]}`)
		},
	)

	channelMapping, errGetChannelMapping := client.GetChannelMapping("token1", "/20131021/lineups/CAN-0000001-X")
	if errGetChannelMapping != nil {
		t.Fatal(errGetChannelMapping)
	}

	if len(channelMapping.Map) != 2 {
		t.Fail()
	}
	if len(channelMapping.Stations) != 2 {
		t.Fail()
	}
	if channelMapping.Metadata.Lineup != "CAN-0000000-X" {
		t.Fail()
	}
}

func TestGetChannelMappingFailsLineupNotFound(t *testing.T) {
	setup()

	mux.HandleFunc("/20131021/lineups/CAN-0000001-X",
		func(w http.ResponseWriter, r *http.Request) {
			testMethod(t, r, "GET")
			testHeader(t, r, "token", "token1")
			fmt.Fprint(w, `{"response":"LINEUP_NOT_FOUND","code":2101,"serverID":"serverid1","message":"Lineup not in account. Add lineup to account before requesting mapping.","datetime":"2014-07-30T04:14:27Z"}`)
		},
	)

	_, errGetChannelMapping := client.GetChannelMapping("token1", "/20131021/lineups/CAN-0000001-X")
	if errGetChannelMapping == nil {
		t.Fail()
	} else if errGetChannelMapping.Error() != "Lineup not in account. Add lineup to account before requesting mapping." {
		t.Fail()
	}
}
