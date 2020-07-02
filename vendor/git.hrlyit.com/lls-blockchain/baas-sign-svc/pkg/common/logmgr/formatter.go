/**
 * @Author: lyszhang
 * @Email: zhangliang@link-logis.com
 * @Date: 2020/6/29 3:59 PM
 */

package logmgr

import (
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"runtime"
)

const (
	defaultTimestampFormat = "2006-01-02 15:04:05"
	FieldKeyUsage          = "usage"
	FieldKeyType           = "type"
	FieldKeyNamespace      = "namespace"
	FieldKeyPodname        = "podname"
	FieldKeyApp            = "app"
	FieldKeyThread         = "thread"
	FieldKeyClass          = "class"
	FieldKeyStack          = "stack_trace"
)

// Usage type
type Usage uint32

const (
	AuditUsage Usage = iota
	ChaincodeUsage
	RuntimeUsage
)

func (usage Usage) MarshalText() ([]byte, error) {
	switch usage {
	case AuditUsage:
		return []byte("audit"), nil
	case ChaincodeUsage:
		return []byte("chaincode"), nil
	case RuntimeUsage:
		return []byte("runtime"), nil
	}
	return nil, fmt.Errorf("not a valid usage %d", usage)
}

// Log type
type LogType uint32

const (
	FabricType LogType = iota
	WebType
	SvcType
)

func (logType LogType) MarshalText() ([]byte, error) {
	switch logType {
	case FabricType:
		return []byte("fabric"), nil
	case WebType:
		return []byte("web"), nil
	case SvcType:
		return []byte("svc"), nil
	}
	return nil, fmt.Errorf("not a valid log type %d", logType)
}

type fieldKey string

// FieldMap allows customization of the key names for default fields.
type FieldMap map[fieldKey]string

func defaultLogFormatter(app string, usage Usage, logType LogType) *jsonFormatter {
	return &jsonFormatter{
		FieldMap: FieldMap{
			log.FieldKeyMsg:  "message",
			log.FieldKeyTime: "timestamp",
		},
		Fields: log.Fields{
			FieldKeyUsage:     usage,
			FieldKeyType:      logType,
			FieldKeyNamespace: GoNamespace,
			FieldKeyPodname:   GoPodname,
			FieldKeyApp:       app,
			FieldKeyThread:    GoID,
			FieldKeyClass:     "",
			FieldKeyStack:     "",
		},
	}
}

func chaincodeLogFormatter(app string, usage Usage, logType LogType) *jsonFormatter {
	return &jsonFormatter{
		DisableMessageAndLevel: true,
		FieldMap: FieldMap{
			log.FieldKeyMsg:  "message",
			log.FieldKeyTime: "timestamp",
		},
		Fields: log.Fields{
			FieldKeyUsage: usage,
			FieldKeyType:  logType,
			FieldKeyApp:   app,
		},
	}
}

func fabricLogFormatter() *jsonFormatter {
	return &jsonFormatter{
		FieldMap: FieldMap{
			log.FieldKeyMsg:  "message",
			log.FieldKeyTime: "ts",
		},
		Fields: log.Fields{
			FieldKeyUsage: "runtime",
			FieldKeyType:  "fabric",
			"name":        "fabric-ca",
			"stacktrace":  "",
		},
	}
}

func (f FieldMap) resolve(key fieldKey) string {
	if k, ok := f[key]; ok {
		return k
	}

	return string(key)
}

// JSONFormatter formats logs into parsable json
type jsonFormatter struct {
	// TimestampFormat sets the format used for marshaling timestamps.
	TimestampFormat string

	// DisableTimestamp allows disabling automatic timestamps in output
	DisableTimestamp bool

	// DataKey allows users to put all the log entry parameters into a nested dictionary at a given key.
	DataKey string

	// FieldMap allows users to customize the names of keys for default fields.
	// As an example:
	// formatter := &JSONFormatter{
	//   	FieldMap: FieldMap{
	// 		 FieldKeyTime:  "@timestamp",
	// 		 FieldKeyLevel: "@level",
	// 		 FieldKeyMsg:   "@message",
	// 		 FieldKeyFunc:  "@caller",
	//    },
	// }
	FieldMap FieldMap

	// DisableMessageAndLevel allows disabling automatic message and level in output
	DisableMessageAndLevel bool

	// CallerPrettyfier can be set by the user to modify the content
	// of the function and file keys in the json data when ReportCaller is
	// activated. If any of the returned value is the empty string the
	// corresponding key will be removed from json fields.
	CallerPrettyfier func(*runtime.Frame) (function string, file string)

	// PrettyPrint will indent all json logs
	PrettyPrint bool

	// WARN:  customize the names of keys for any fields
	Fields log.Fields
}

// Format renders a single log entry
func (f *jsonFormatter) Format(entry *log.Entry) ([]byte, error) {
	data := make(log.Fields, len(entry.Data)+4+len(f.Fields))
	for k, v := range entry.Data {
		switch v := v.(type) {
		case error:
			// Otherwise errors are ignored by `encoding/json`
			// https://github.com/sirupsen/logrus/issues/137
			data[k] = v.Error()
		default:
			data[k] = v
		}
	}

	if f.DataKey != "" {
		newData := make(log.Fields, 4)
		newData[f.DataKey] = data
		data = newData
	}

	prefixFieldClashes(data, f.FieldMap, entry.HasCaller())

	timestampFormat := f.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = defaultTimestampFormat
	}

	if !f.DisableTimestamp {
		data[f.FieldMap.resolve(log.FieldKeyTime)] = entry.Time.Format(timestampFormat)
	}

	if !f.DisableMessageAndLevel {
		data[f.FieldMap.resolve(log.FieldKeyMsg)] = entry.Message
		data[f.FieldMap.resolve(log.FieldKeyLevel)] = entry.Level.String()
	}

	if entry.HasCaller() {
		funcVal := entry.Caller.Function
		fileVal := fmt.Sprintf("%s:%d", entry.Caller.File, entry.Caller.Line)
		if f.CallerPrettyfier != nil {
			funcVal, fileVal = f.CallerPrettyfier(entry.Caller)
		}
		if funcVal != "" {
			data[f.FieldMap.resolve(log.FieldKeyFunc)] = funcVal
		}
		if fileVal != "" {
			data[f.FieldMap.resolve(log.FieldKeyFile)] = fileVal
		}
	}

	// customize key & value
	for key, value := range f.Fields {
		switch key {
		case FieldKeyThread:
			out, ok := value.(func() int)
			if !ok {
				return nil, fmt.Errorf("FieldKeyThread func type mismatch")
			}
			data[key] = out()
		case FieldKeyPodname, FieldKeyNamespace:
			out, ok := value.(func() string)
			if !ok {
				return nil, fmt.Errorf("FieldKeyThread func type mismatch")
			}
			data[key] = out()
		default:
			data[key] = value
		}
	}

	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	encoder := json.NewEncoder(b)
	if f.PrettyPrint {
		encoder.SetIndent("", "  ")
	}
	if err := encoder.Encode(data); err != nil {
		return nil, fmt.Errorf("failed to marshal fields to JSON, %v", err)
	}

	return b.Bytes(), nil
}

// This is to not silently overwrite `time`, `msg`, `func` and `level` fields when
// dumping it. If this code wasn't there doing:
//
//  logrus.WithField("level", 1).Info("hello")
//
// Would just silently drop the user provided level. Instead with this code
// it'll logged as:
//
//  {"level": "info", "fields.level": 1, "msg": "hello", "time": "..."}
//
// It's not exported because it's still using Data in an opinionated way. It's to
// avoid code duplication between the two default formatters.
func prefixFieldClashes(data log.Fields, fieldMap FieldMap, reportCaller bool) {
	timeKey := fieldMap.resolve(log.FieldKeyTime)
	if t, ok := data[timeKey]; ok {
		data["fields."+timeKey] = t
		delete(data, timeKey)
	}

	msgKey := fieldMap.resolve(log.FieldKeyMsg)
	if m, ok := data[msgKey]; ok {
		data["fields."+msgKey] = m
		delete(data, msgKey)
	}

	levelKey := fieldMap.resolve(log.FieldKeyLevel)
	if l, ok := data[levelKey]; ok {
		data["fields."+levelKey] = l
		delete(data, levelKey)
	}

	logrusErrKey := fieldMap.resolve(log.FieldKeyLogrusError)
	if l, ok := data[logrusErrKey]; ok {
		data["fields."+logrusErrKey] = l
		delete(data, logrusErrKey)
	}

	// If reportCaller is not set, 'func' will not conflict.
	if reportCaller {
		funcKey := fieldMap.resolve(log.FieldKeyFunc)
		if l, ok := data[funcKey]; ok {
			data["fields."+funcKey] = l
		}
		fileKey := fieldMap.resolve(log.FieldKeyFile)
		if l, ok := data[fileKey]; ok {
			data["fields."+fileKey] = l
		}
	}
}
