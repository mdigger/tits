package main

type MXA struct {
	mxa *MXAClient
}

func (s *MXA) Add(obj User, id *string) (err error) {
	*id, err = s.mxa.Add(obj)
	return
}

func (s *MXA) Update(obj User, ok *bool) (err error) {
	err = s.mxa.Update(obj)
	*ok = err == nil
	return
}

func (s *MXA) Delete(id string, ok *bool) (err error) {
	err = s.mxa.Delete(id)
	*ok = err == nil
	return
}
