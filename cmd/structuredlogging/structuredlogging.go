package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"time"
)

// https://cloud.google.com/logging/docs/structured-logging
type logEntry struct {
	Severity       string            `json:"severity"`
	Message        interface{}       `json:"message"`
	HttpRequest    *httpRequest      `json:"httpRequest,omitempty"`
	Timestamp      time.Time         `json:"timestamp"`
	Labels         map[string]string `json:"logging.googleapis.com/labels,omitempty"`
	Operation      *operation        `json:"logging.googleapis.com/operation,omitempty"`
	SourceLocation *sourceLocation   `json:"logging.googleapis.com/sourceLocation,omitempty"`
	SpanID         string            `json:"logging.googleapis.com/spanId,omitempty"`
	TraceID        string            `json:"logging.googleapis.com/trace,omitempty"`
	TraceSampled   bool              `json:"logging.googleapis.com/trace_sampled,omitempty"`
}

type httpRequest struct {
	RequestMethod                  string `json:"requestMethod,omitempty"`
	RequestUrl                     string `json:"requestUrl,omitempty"`
	RequestSize                    string `json:"requestSize,omitempty"`
	Status                         int    `json:"status,omitempty"`
	ResponseSize                   string `json:"responseSize,omitempty"`
	UserAgent                      string `json:"userAgent,omitempty"`
	RemoteIp                       string `json:"remoteIp,omitempty"`
	ServerIp                       string `json:"serverIp,omitempty"`
	Referer                        string `json:"referer,omitempty"`
	Latency                        string `json:"latency,omitempty"`
	CacheLookup                    bool   `json:"cacheLookup,omitempty"`
	CacheHit                       bool   `json:"cacheHit,omitempty"`
	CacheValidatedWithOriginServer bool   `json:"cacheValidatedWithOriginServer,omitempty"`
	CacheFillBytes                 string `json:"cacheFillBytes,omitempty"`
	Protocol                       string `json:"protocol,omitempty"`
}

type operation struct {
	Id       string `json:"id,omitempty"`
	Producer string `json:"producer,omitempty"`
	First    string `json:"first,omitempty"`
	Last     string `json:"last,omitempty"`
}

type sourceLocation struct {
	File     string `json:"file,omitempty"`
	Line     string `json:"line,omitempty"`
	Function string `json:"function,omitempty"`
}

func info(r *http.Request, message interface{}, projectID string) {
	get := r.Header.Get("X-Cloud-Trace-Context")
	traceID, spanID, traceSampled := deconstructXCloudTraceContext(get)
	traceID = fmt.Sprintf("projects/%s/traces/%s", projectID, traceID)
	entry := logEntry{
		Severity: "INFO",
		Message:  message,
		HttpRequest: &httpRequest{
			RequestMethod: r.Method,
			RequestUrl:    r.URL.String(),
			UserAgent:     r.UserAgent(),
			RemoteIp:      r.RemoteAddr,
			Referer:       r.Referer(),
		},
		Timestamp:    time.Now(),
		Labels:       map[string]string{"labels": "rock"},
		SpanID:       spanID,
		TraceID:      traceID,
		TraceSampled: traceSampled,
	}
	writelog(&entry)
}

func writelog(entry *logEntry) {
	if err := json.NewEncoder(os.Stderr).Encode(entry); err != nil {
		fmt.Printf("failure to write structured log entry: %v", err)
	}
}

// taken from https://github.com/googleapis/google-cloud-go/blob/master/logging/logging.go#L774
var reCloudTraceContext = regexp.MustCompile(
	// Matches on "TRACE_ID"
	`([a-f\d]+)?` +
		// Matches on "/SPAN_ID"
		`(?:/([a-f\d]+))?` +
		// Matches on ";0=TRACE_TRUE"
		`(?:;o=(\d))?`)

func deconstructXCloudTraceContext(s string) (traceID, spanID string, traceSampled bool) {
	// As per the format described at https://cloud.google.com/trace/docs/setup#force-trace
	//    "X-Cloud-Trace-Context: TRACE_ID/SPAN_ID;o=TRACE_TRUE"
	// for example:
	//    "X-Cloud-Trace-Context: 105445aa7843bc8bf206b120001000/1;o=1"
	//
	// We expect:
	//   * traceID (optional): 			"105445aa7843bc8bf206b120001000"
	//   * spanID (optional):       	"1"
	//   * traceSampled (optional): 	true
	matches := reCloudTraceContext.FindStringSubmatch(s)

	traceID, spanID, traceSampled = matches[1], matches[2], matches[3] == "1"

	if spanID == "0" {
		spanID = ""
	}

	return
}

func debug(r *http.Request, message interface{}, projectID string) {
	get := r.Header.Get("X-Cloud-Trace-Context")
	traceID, spanID, traceSampled := deconstructXCloudTraceContext(get)
	traceID = fmt.Sprintf("projects/%s/traces/%s", projectID, traceID)
	entry := logEntry{
		Severity: "DEBUG",
		Message:  message,
		HttpRequest: &httpRequest{
			RequestMethod: r.Method,
			RequestUrl:    r.URL.String(),
			UserAgent:     r.UserAgent(),
			RemoteIp:      r.RemoteAddr,
			Referer:       r.Referer(),
		},
		Timestamp:    time.Now(),
		Labels:       map[string]string{"labels": "rock"},
		SpanID:       spanID,
		TraceID:      traceID,
		TraceSampled: traceSampled,
	}
	writelog(&entry)
}

func notice(r *http.Request, message interface{}, projectID string) {
	get := r.Header.Get("X-Cloud-Trace-Context")
	traceID, spanID, traceSampled := deconstructXCloudTraceContext(get)
	traceID = fmt.Sprintf("projects/%s/traces/%s", projectID, traceID)
	entry := logEntry{
		Severity: "NOTICE",
		Message:  message,
		HttpRequest: &httpRequest{
			RequestMethod: r.Method,
			RequestUrl:    r.URL.String(),
			UserAgent:     r.UserAgent(),
			RemoteIp:      r.RemoteAddr,
			Referer:       r.Referer(),
		},
		Timestamp:    time.Now(),
		Labels:       map[string]string{"labels": "rock"},
		SpanID:       spanID,
		TraceID:      traceID,
		TraceSampled: traceSampled,
	}
	writelog(&entry)
}

func warning(r *http.Request, message interface{}, projectID string) {
	get := r.Header.Get("X-Cloud-Trace-Context")
	traceID, spanID, traceSampled := deconstructXCloudTraceContext(get)
	traceID = fmt.Sprintf("projects/%s/traces/%s", projectID, traceID)
	entry := logEntry{
		Severity: "WARNING",
		Message:  message,
		HttpRequest: &httpRequest{
			RequestMethod: r.Method,
			RequestUrl:    r.URL.String(),
			UserAgent:     r.UserAgent(),
			RemoteIp:      r.RemoteAddr,
			Referer:       r.Referer(),
		},
		Timestamp:    time.Now(),
		Labels:       map[string]string{"labels": "rock"},
		SpanID:       spanID,
		TraceID:      traceID,
		TraceSampled: traceSampled,
	}
	writelog(&entry)
}

func errorl(r *http.Request, message interface{}, projectID string) {
	get := r.Header.Get("X-Cloud-Trace-Context")
	traceID, spanID, traceSampled := deconstructXCloudTraceContext(get)
	traceID = fmt.Sprintf("projects/%s/traces/%s", projectID, traceID)
	entry := logEntry{
		Severity: "ERROR",
		Message:  message,
		HttpRequest: &httpRequest{
			RequestMethod: r.Method,
			RequestUrl:    r.URL.String(),
			UserAgent:     r.UserAgent(),
			RemoteIp:      r.RemoteAddr,
			Referer:       r.Referer(),
		},
		Timestamp:    time.Now(),
		Labels:       map[string]string{"labels": "rock"},
		SpanID:       spanID,
		TraceID:      traceID,
		TraceSampled: traceSampled,
	}
	writelog(&entry)
}

func critical(r *http.Request, message interface{}, projectID string) {
	get := r.Header.Get("X-Cloud-Trace-Context")
	traceID, spanID, traceSampled := deconstructXCloudTraceContext(get)
	traceID = fmt.Sprintf("projects/%s/traces/%s", projectID, traceID)
	entry := logEntry{
		Severity: "CRITICAL",
		Message:  message,
		HttpRequest: &httpRequest{
			RequestMethod: r.Method,
			RequestUrl:    r.URL.String(),
			UserAgent:     r.UserAgent(),
			RemoteIp:      r.RemoteAddr,
			Referer:       r.Referer(),
		},
		Timestamp:    time.Now(),
		Labels:       map[string]string{"labels": "rock"},
		SpanID:       spanID,
		TraceID:      traceID,
		TraceSampled: traceSampled,
	}
	writelog(&entry)
}

func alert(r *http.Request, message interface{}, projectID string) {
	get := r.Header.Get("X-Cloud-Trace-Context")
	traceID, spanID, traceSampled := deconstructXCloudTraceContext(get)
	traceID = fmt.Sprintf("projects/%s/traces/%s", projectID, traceID)
	entry := logEntry{
		Severity: "ALERT",
		Message:  message,
		HttpRequest: &httpRequest{
			RequestMethod: r.Method,
			RequestUrl:    r.URL.String(),
			UserAgent:     r.UserAgent(),
			RemoteIp:      r.RemoteAddr,
			Referer:       r.Referer(),
		},
		Timestamp:    time.Now(),
		Labels:       map[string]string{"labels": "rock"},
		SpanID:       spanID,
		TraceID:      traceID,
		TraceSampled: traceSampled,
	}
	writelog(&entry)
}

func emergency(r *http.Request, message interface{}, projectID string) {
	get := r.Header.Get("X-Cloud-Trace-Context")
	traceID, spanID, traceSampled := deconstructXCloudTraceContext(get)
	traceID = fmt.Sprintf("projects/%s/traces/%s", projectID, traceID)
	entry := logEntry{
		Severity: "EMERGENCY",
		Message:  message,
		HttpRequest: &httpRequest{
			RequestMethod: r.Method,
			RequestUrl:    r.URL.String(),
			UserAgent:     r.UserAgent(),
			RemoteIp:      r.RemoteAddr,
			Referer:       r.Referer(),
		},
		Timestamp:    time.Now(),
		Labels:       map[string]string{"labels": "rock"},
		SpanID:       spanID,
		TraceID:      traceID,
		TraceSampled: traceSampled,
	}
	writelog(&entry)
}
