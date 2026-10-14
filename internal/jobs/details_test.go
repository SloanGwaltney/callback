package jobs

import (
	"reflect"
	"testing"
)

// Each Details method copies every detail, so a change that starts from what's saved, like the
// agent command line's `job update`, never clears a detail added later. These fill every field of
// the saved record, then check each detail matches the record's field of the same name.

func TestJobDetailsCopiesEveryDetail(t *testing.T) {
	var job Job
	fillFields(&job)
	checkDetailsMatch(t, job, job.Details())
}

func TestContactDetailsCopiesEveryDetail(t *testing.T) {
	var contact Contact
	fillFields(&contact)
	checkDetailsMatch(t, contact, contact.Details())
}

func TestInterviewDetailsCopiesEveryDetail(t *testing.T) {
	var interview Interview
	fillFields(&interview)
	checkDetailsMatch(t, interview, interview.Details())
}

// fillFields gives every string, number, flag, and pointer to a number in *v a value that isn't
// zero, each different, so a detail copied from the wrong field is caught too.
func fillFields(v any) {
	s := reflect.ValueOf(v).Elem()
	for i := range s.NumField() {
		f := s.Field(i)
		switch f.Kind() {
		case reflect.String:
			f.SetString(s.Type().Field(i).Name)
		case reflect.Int, reflect.Int64:
			f.SetInt(int64(i + 1))
		case reflect.Bool:
			f.SetBool(true)
		case reflect.Pointer:
			if f.Type().Elem().Kind() == reflect.Int {
				n := reflect.New(f.Type().Elem())
				n.Elem().SetInt(int64(i + 1))
				f.Set(n)
			}
		}
	}
}

// checkDetailsMatch checks that every field of details is set, and equals the field of the same
// name on record.
func checkDetailsMatch(t *testing.T, record, details any) {
	t.Helper()
	r := reflect.ValueOf(record)
	d := reflect.ValueOf(details)
	for i := range d.NumField() {
		name := d.Type().Field(i).Name
		got := d.Field(i)
		want := r.FieldByName(name)
		switch {
		case !want.IsValid():
			t.Errorf("%s has no %s field to copy", r.Type().Name(), name)
		case got.IsZero():
			t.Errorf("%s.Details() leaves %s out", r.Type().Name(), name)
		case !reflect.DeepEqual(got.Interface(), want.Interface()):
			t.Errorf("%s.Details().%s = %v, want %v", r.Type().Name(), name, got.Interface(), want.Interface())
		}
	}
}
