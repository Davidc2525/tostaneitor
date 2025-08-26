package main

type Fake struct{}
type FakeConn struct{}

func (t *Fake) Dial() *FakeConn {
	return &FakeConn{}
}

func (f *FakeConn) Read(msg []byte) (n int, err error) {
	return
}

func (f *FakeConn) Close() {

}
