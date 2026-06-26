package sim

import (
	"encoding/json"
	"fmt"
	"time"
)

type LockDevice struct {
	ID    string
	State string
}

type Simulator struct {
	locks map[string]*LockDevice
}

func NewSimulator() *Simulator {
	return &Simulator{
		locks: map[string]*LockDevice{
			"lock-main": {ID: "lock-main", State: "locked"},
		},
	}
}

func (s *Simulator) MotionEvent(zone string) map[string]interface{} {
	return map[string]interface{}{
		"type":            "iot_event",
		"device_id":       "sensor-motion-01",
		"device_type":     "motion_sensor",
		"zone":            zone,
		"payload_summary": fmt.Sprintf("motion detected %s", zone),
		"timestamp":       time.Now().Unix(),
	}
}

func (s *Simulator) Telemetry(deviceID string, value float64) map[string]interface{} {
	return map[string]interface{}{
		"type":      "iot_telemetry",
		"device_id": deviceID,
		"value":     value,
		"unit":      "kwh",
		"timestamp": time.Now().Unix(),
	}
}

func (s *Simulator) LockCommand(deviceID, action string, durationSec int) (map[string]interface{}, error) {
	lock, ok := s.locks[deviceID]
	if !ok {
		lock = &LockDevice{ID: deviceID, State: "locked"}
		s.locks[deviceID] = lock
	}
	switch action {
	case "unlock":
		lock.State = "unlocked"
	case "lock":
		lock.State = "locked"
	case "status":
	default:
		return nil, fmt.Errorf("unknown action")
	}
	res := map[string]interface{}{
		"type":           "task_result",
		"device_id":      deviceID,
		"state":          lock.State,
		"previous_state": opposite(lock.State),
	}
	if action == "unlock" && durationSec > 0 {
		res["auto_relock_at"] = time.Now().Add(time.Duration(durationSec) * time.Second).Unix()
	}
	return res, nil
}

func (s *Simulator) GetLockState(deviceID string) string {
	if l, ok := s.locks[deviceID]; ok {
		return l.State
	}
	return "locked"
}

func opposite(state string) string {
	if state == "locked" {
		return "unlocked"
	}
	return "locked"
}

func (s *Simulator) ToJSON(m map[string]interface{}) string {
	b, _ := json.Marshal(m)
	return string(b)
}

// Float returns a small varying float for simulated telemetry demos.
func (s *Simulator) Float() float64 {
	return float64(time.Now().Unix()%100) / 10.0
}
