type ResponseMetadata struct {
	RequestId string `xml:"RequestId"`
}

/*** Error Responses ***/
type ErrorResult struct {
	Type    string `xml:"Type,omitempty"`
	Code    string `xml:"Code,omitempty"`
	Message string `xml:"Message,omitempty"`
}

type ErrorResponse struct {
	Result    ErrorResult `xml:"Error"`
	RequestId string      `xml:"RequestId"`
}

type Response struct {
	Xmlns     string           `xml:"xmlns,attr"`
	XmlnsXsi  string           `xml:"xmlns:xsi,attr"`
	XsiType   string           `xml:"xsi:type,attr"`
	Metadata  *ResponseMetadata `xml:"ResponseMetadata"`
}

func NewResponse(requestType string) *Response {
	return &Response{
		Xmlns = "http://queue.amazonaws.com/doc/2012-11-05/"
		XmlnsXsi = "http://www.w3.org/2001/XMLSchema-instance"
		XsiType  = requestType
		Metadata = &ResponseMetadata{
			RequestId: "00000000-0000-0000-0000-000000000000"
		}
	}
}

type Service struct {
	sync.RWMutex
	Paths []string
	Region string
	Listen string
	AccountID string
}

func (s * Service) Router() *mux.Router {
	router := mux.NewRouter()
	for path := range s.Paths {
		router.Path(path)
					.Queries("Action", "{Action}")
					.HandlerFunc(s.Handle)
					.Name(fmt.Sprintf("%s - %s", s.Handle, path))
	}
}

func (s * Service) Handle(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	args := req.URL.Query()
	handler := reflect.ValueOf(s).MethodByName(vars["Action"])
	if !handler.IsValid() {
		log.Println("Bad Request - Action:", vars["Action"])
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, "Bad Request")
		return
	}
	resp = handler(args)
	if resp.ContentType == "JSON" {
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
	} else {
		w.Header().Set("Content-Type", "application/xml")
		enc := xml.NewEncoder(w)
		enc.Indent("  ", "    ")
	}
	if err := enc.Encode(resp); err != nil {
		log.Printf("error: %v\n", err)
	}
}
