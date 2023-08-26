package utc

// import "time"

// type Time struct {
// 	t time.Time
// }

// func Now() Time {
// 	return Time{t: time.Now().UTC()}
// }

// func Since(t Time) time.Duration {
// 	return Now().Sub(t)
// }

// func Date(year int, month time.Month, day, hour, minute, second, nsec int) Time {
// 	t := time.Date(year, month, day, hour, minute, second, nsec, time.UTC)
// 	return Time{t}
// }

// func (t Time) Add(d time.Duration) Time {
// 	return Time{t: t.t.Add(d)}
// }

// func (t Time) Sub(u Time) time.Duration {
// 	return t.t.Sub(u.t)
// }

// func (t Time) Format(layout string) string {
// 	return t.t.Format(layout)
// }

// func (t Time) After(u Time) bool {
// 	return t.t.After(u.t)
// }

// func (t Time) Date() (year int, month time.Month, day int) {
// 	return t.t.Date()
// }

// func (t Time) Milli() int64 {
// 	return t.t.UnixMilli()
// }

// func (t Time) Micro() int64 {
// 	return t.t.UnixMicro()
// }

// func UnixMilli(msec int64) Time {
// 	return Time{t: time.UnixMilli(msec).UTC()}
// }
