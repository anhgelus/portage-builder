package proto

import (
	"context"
	"testing"

	"pgregory.net/rapid"
)

func TestClient(t *testing.T) {
	s := NewServer(nil, &dummyServer{}, 1024*1024*1024)
	ctx := context.Background()
	rapid.Check(t, func(t *rapid.T) {
		dualCom := dummyDualCom{newDummyCom(), newDummyCom()}
		defer dualCom.Close()
		s.ReadWriteCloser = dualCom.server
		errc := setupAutoServer(s, &dualCom)
		cl, err := NewClient(context.Background(), dualCom.client)
		if err != nil {
			t.Fatal(err)
		}
		go func() {
			for err := range errc {
				t.Error(err)
			}
		}()
		req := genRequest().Draw(t, "req")
		switch v := req.want.(type) {
		case BuildArg:
			err = cl.RequestBuild(ctx, v)
		case CfgArg:
			err = cl.RequestConfig(ctx, v)
		case SendArg:
			_, err = cl.RequestSend(ctx, v)
		case PartArg:
			err = cl.RequestPart(ctx, v)
		}
		if err != nil {
			t.Error(err)
		}
	})
}
