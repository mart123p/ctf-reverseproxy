package reverseproxy

import "github.com/mart123p/ctf-reverseproxy/pkg/cbroadcast"

const BProxyMetricTime = "proxy:metric:time"
const BSize = 5

func (rp *ReverseProxy) Register() {
	cbroadcast.Register(BProxyMetricTime, BSize)
}
