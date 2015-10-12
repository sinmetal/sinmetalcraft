package sinmetalcraft

import (
	"testing"
)

func TestPubSubBodyDecode(t *testing.T) {
	json := `{"message":{"data":"eyJtZXRhZGF0YSI6eyJwcm9qZWN0SWQiOiJzaW5tZXRhbGNyYWZ0Iiwic2VydmljZU5hbWUiOiJjb21wdXRlLmdvb2dsZWFwaXMuY29tIiwiem9uZSI6ImFzaWEtZWFzdDEtYiIsImxhYmVscyI6eyJjb21wdXRlLmdvb2dsZWFwaXMuY29tL3Jlc291cmNlX2lkIjoiMTYxNjg5ODI0NjY1MjQ5MTY0MjYiLCJjb21wdXRlLmdvb2dsZWFwaXMuY29tL3Jlc291cmNlX3R5cGUiOiJpbnN0YW5jZSJ9LCJ0aW1lc3RhbXAiOiIyMDE1LTEwLTEyVDA4OjE1OjU0WiJ9LCJpbnNlcnRJZCI6IjIwMTUtMTAtMTJ8MDE6MTU6NTcuMDE4MzM3LTA3fDEwLjE4OC40MC4xNDF8LTIzNDEwNTI4NiIsImxvZyI6Im1pbmVjcmFmdCIsInN0cnVjdFBheWxvYWQiOnsibG9nIjoiU3RhcnRpbmcgbWluZWNyYWZ0IHNlcnZlciB2ZXJzaW9uIDEuOC44In19","attributes":{"compute.googleapis.com/resource_id":"16168982466524916426","compute.googleapis.com/resource_type":"instance"},"message_id":"4258433911387"},"subscription":"projects/sinmetalcraft/subscriptions/gae"}`

	var psb PubSubBody
	err := psb.Decode([]byte(json))
	if err != nil {
		t.Fatalf("Pub Sub Body Decode Error: %v", err)
	}
	t.Logf("Pub Sub Body = %v", psb)

	var psd PubSubData
	err = psd.Decode(psb.Message.Data)
	if err != nil {
		t.Fatalf("Pub Sub Data Decode Error: %v, data = %s", err, psb.Message.Data)
	}
	t.Logf("Pub Sub Data = %v", psd)
}
