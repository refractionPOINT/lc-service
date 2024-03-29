package service

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v2"
)

type Job struct {
	isNew   bool
	id      string
	cause   string
	sensors []string

	start int64
	end   int64

	entries []JobEntry
}

type JobEntry struct {
	ts          int64
	msg         string
	attachments []JobAttachment
	isImportant bool
}

type JobAttachment interface {
	ToJSON() map[string]interface{}
}

type hexDumpAttachment struct {
	caption string
	data    string
}

type yamlAttachment struct {
	caption string
	data    string
}

type jsonAttachment struct {
	caption string
	data    string
}

type tableAttachment struct {
	caption string
	headers []string
	rows    [][]string
}

func getMSTimestamp() int64 {
	return time.Now().Unix() * 1000
}

func NewJob(jobID ...string) *Job {
	j := &Job{}
	if len(jobID) == 0 {
		j.isNew = true
		j.id = uuid.New().String()
		j.start = getMSTimestamp()
	} else {
		j.id = jobID[0]
	}
	return j
}

func (j *Job) AddSensor(sensorID string) {
	j.sensors = append(j.sensors, sensorID)
}

func (j *Job) SetCause(cause string) {
	j.cause = cause
}

func (j *Job) Close() {
	j.end = getMSTimestamp()
}

func (j Job) GetID() string {
	return j.id
}

func (j *Job) Narrate(message string, isImportant bool, attachments ...JobAttachment) {
	e := JobEntry{
		ts:          getMSTimestamp(),
		msg:         message,
		isImportant: isImportant,
		attachments: attachments,
	}
	j.entries = append(j.entries, e)
}

func (j Job) ToJSON() map[string]interface{} {
	d := map[string]interface{}{
		"id":   j.id,
		"hist": []map[string]interface{}{},
	}
	if j.start != 0 {
		d["start"] = j.start
	}
	if j.end != 0 {
		d["end"] = j.end
	}
	if j.cause != "" {
		d["cause"] = j.cause
	}
	if len(j.sensors) != 0 {
		d["sid"] = j.sensors
	}
	for _, e := range j.entries {
		d["hist"] = append(d["hist"].([]map[string]interface{}), e.ToJSON())
	}
	return d
}

func (j Job) MarshalJSON() ([]byte, error) {
	return json.Marshal(j.ToJSON())
}

func (e JobEntry) ToJSON() map[string]interface{} {
	a := []map[string]interface{}{}
	for _, at := range e.attachments {
		a = append(a, at.ToJSON())
	}
	d := map[string]interface{}{
		"ts":           e.ts,
		"msg":          e.msg,
		"attachments":  a,
		"is_important": e.isImportant,
	}
	return d
}

func NewHexDumpAttachment(caption string, data []byte) JobAttachment {
	h := hexDumpAttachment{
		caption: caption,
		data:    base64.StdEncoding.EncodeToString(data),
	}
	return &h
}

func (h hexDumpAttachment) ToJSON() map[string]interface{} {
	return map[string]interface{}{
		"att_type": "hex_dump",
		"caption":  h.caption,
		"data":     h.data,
	}
}

func NewYamlAttachment(caption string, data interface{}) JobAttachment {
	y, err := yaml.Marshal(data)
	if err != nil {
		y = []byte(fmt.Sprintf("%+v", data))
	}
	h := yamlAttachment{
		caption: caption,
		data:    string(y),
	}
	return &h
}

func (h yamlAttachment) ToJSON() map[string]interface{} {
	return map[string]interface{}{
		"att_type": "yaml",
		"caption":  h.caption,
		"data":     h.data,
	}
}

func NewJSONAttachment(caption string, data interface{}) JobAttachment {
	y, err := json.Marshal(data)
	if err != nil {
		y = []byte(fmt.Sprintf("%+v", data))
	}
	h := jsonAttachment{
		caption: caption,
		data:    string(y),
	}
	return &h
}

func (h jsonAttachment) ToJSON() map[string]interface{} {
	return map[string]interface{}{
		"att_type": "yaml",
		"caption":  h.caption,
		"data":     h.data,
	}
}

func NewTableAttachment(caption string, headers []string, rows [][]string) JobAttachment {
	h := tableAttachment{
		caption: caption,
		headers: headers,
		rows:    rows,
	}
	return &h
}

func (h tableAttachment) ToJSON() map[string]interface{} {
	return map[string]interface{}{
		"att_type": "table",
		"caption":  h.caption,
		"headers":  h.headers,
		"rows":     h.rows,
	}
}

// TODO add other attachment types.
