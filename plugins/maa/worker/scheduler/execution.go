package scheduler

import "sync"

// All plugin commands and schedules use this execution lock.
var ExecutionMu sync.Mutex

// Compatibility with the custom GUI runner's scheduler suspension flag. This
// worker contains only the persistent GUI scheduler, not the legacy task engine.
var ScheduleData = struct{ IsStop bool }{}
