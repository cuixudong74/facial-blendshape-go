package store

import "errors"

func errNilTarget() error       { return errors.New("target is nil") }
func errDuplicateTarget() error { return errors.New("duplicate target name in batch") }
