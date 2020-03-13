package pay

import (
	"fmt"
	"testing"

	"github.com/aimo-x/wechat/util"
)

func TestSign(T *testing.T) {
	template := "appId=%s&nonceStr=%s&package=%s&signType=%s&timeStamp=%s&key=%s"
	str := fmt.Sprintf(template, "wx016b7f8177b8a007", "lwRVpZGwsCPOfohV2CrXtRroDb9vzRPh", "prepay_id=wx11013410710840fc916cf5a91113520800", "MD5", "1565458450", "1ebbd6188e79ed64fc2c4f957a988a5b")
	sign := util.MD5Sum(str)
	fmt.Println(str, "sign: ", sign)
}
