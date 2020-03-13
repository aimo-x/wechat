package pay

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/aimo-x/wechat/context"
	"github.com/aimo-x/wechat/util"
)

var payGateway = "https://api.mch.weixin.qq.com/pay/unifiedorder"
var orderQueryURI = "https://api.mch.weixin.qq.com/pay/orderquery"
var orderQueryURI2 = "https://api2.mch.weixin.qq.com/pay/orderquery"

// Pay struct extends context
type Pay struct {
	*context.Context
}

// UnifiedorderParams was NEEDED when request unifiedorder
// 统一下单 传入的参数，用于生成 prepay_id 的必需参数  FeeType [本平台自增加]= HKD
type UnifiedorderParams struct {
	TotalFee     string // TotalFee 订单总金额，单位为分
	CreateIP     string // CreateIP 客户端IP 支持IPV6
	Body         string // Body 商品描述 长度	128
	FeeType      string // 标价币种	符合ISO 4217标准的三位字母代码 3
	OutTradeNo   string // 商户订单号码，唯一
	OpenID       string // openid 收取获取
	PayNotifyURL string //通知地址
}

// JSAPIParams 是传出用于 JSAPIConfig 用的参数
type JSAPIParams struct {
	AppID     string
	Timestamp int64
	NonceStr  string
	Package   string
	SignType  string
	Sign      string
}

// payResult 是 unifie order 接口的返回
type payResult struct {
	ReturnCode string `xml:"return_code"`
	ReturnMsg  string `xml:"return_msg"`
	AppID      string `xml:"appid,omitempty"`
	MchID      string `xml:"mch_id,omitempty"`
	NonceStr   string `xml:"nonce_str,omitempty"`
	Sign       string `xml:"sign,omitempty"`
	ResultCode string `xml:"result_code,omitempty"`
	TradeType  string `xml:"trade_type,omitempty"`
	PrePayID   string `xml:"prepay_id,omitempty"`
	CodeURL    string `xml:"code_url,omitempty"`
	ErrCode    string `xml:"err_code,omitempty"`
	ErrCodeDes string `xml:"err_code_des,omitempty"`
}

//payRequest 接口请求参数
type payRequest struct {
	AppID          string `xml:"appid"`
	MchID          string `xml:"mch_id"`
	DeviceInfo     string `xml:"device_info,omitempty"`
	NonceStr       string `xml:"nonce_str"`
	Sign           string `xml:"sign"`
	SignType       string `xml:"sign_type,omitempty"`
	Body           string `xml:"body"`
	Detail         string `xml:"detail,omitempty"`
	Attach         string `xml:"attach,omitempty"`      //附加数据
	OutTradeNo     string `xml:"out_trade_no"`          //商户订单号
	FeeType        string `xml:"fee_type,omitempty"`    //标价币种
	TotalFee       string `xml:"total_fee"`             //标价金额
	SpbillCreateIP string `xml:"spbill_create_ip"`      //终端IP
	TimeStart      string `xml:"time_start,omitempty"`  //交易起始时间
	TimeExpire     string `xml:"time_expire,omitempty"` //交易结束时间
	GoodsTag       string `xml:"goods_tag,omitempty"`   //订单优惠标记
	NotifyURL      string `xml:"notify_url"`            //通知地址
	TradeType      string `xml:"trade_type"`            //交易类型
	ProductID      string `xml:"product_id,omitempty"`  //商品ID
	LimitPay       string `xml:"limit_pay,omitempty"`   //
	OpenID         string `xml:"openid,omitempty"`      //用户标识
	SceneInfo      string `xml:"scene_info,omitempty"`  //场景信息
}

// OrderQueryParams 用以查询订单的传入参数
type OrderQueryParams struct {
	TransactionID string `xml:"transaction_id,omitempty"` // 微信订单号 TransactionID and OutTradeNo 二选1
	OutTradeNo    string `xml:"out_trade_no,omitempty"`   // 商户系统内部订单号，要求32个字符内，只能是数字、大小写字母_-|*@ ，且在同一个商户号下唯一。
}

// OrderQueryRequest 发起订单查询的结构体
type OrderQueryRequest struct {
	AppID         string `xml:"appid"`                    // 微信分配的公众账号ID
	MchID         string `xml:"mch_id"`                   // 微信支付分配的商户号
	TransactionID string `xml:"transaction_id,omitempty"` // 微信订单号 TransactionID and OutTradeNo 二选1
	OutTradeNo    string `xml:"out_trade_no,omitempty"`   // 商户系统内部订单号，要求32个字符内，只能是数字、大小写字母_-|*@ ，且在同一个商户号下唯一。
	NonceStr      string `xml:"nonce_str"`                // 随机字符串，不长于32位。推荐随机数生成算法
	Sign          string `xml:"sign"`                     // 签名，详见签名生成算法
	SignType      string `xml:"sign_type,omitempty"`      // 签名类型
}

// OrderQueryResult 查询订单的返回结果 暂时不支持 查询 代金券类型，代金券ID，单个代金券支付金额
type OrderQueryResult struct {
	ReturnCode string `xml:"return_code"` // UCCESS/FAIL	此字段是通信标识，非交易标识，交易是否成功需要查看trade_state来判断
	ReturnMsg  string `xml:"return_msg"`  // 当return_code为FAIL时返回信息为错误原因 ，例如 签名失败 参数格式校验错误

	AppID      string `xml:"appid,omitempty"`        // 微信分配的公众账号ID
	MchID      string `xml:"mch_id,omitempty"`       // 微信支付分配的商户号
	NonceStr   string `xml:"nonce_str,omitempty"`    // 随机字符串，不长于32位。推荐随机数生成算法
	Sign       string `xml:"sign,omitempty"`         // 签名，详见签名生成算法
	ResultCode string `xml:"result_code,omitempty"`  // SUCCESS/FAIL
	ErrCode    string `xml:"err_code,omitempty"`     // 当result_code为FAIL时返回错误代码，详细参见下文错误列表
	ErrCodeDes string `xml:"err_code_des,omitempty"` // 当result_code为FAIL时返回错误描述，详细参见下文错误列表

	TradeState string `xml:"trade_state,omitempty"` // SUCCESS—支付成功 REFUND—转入退款NOTPAY—未支付CLOSED—已关闭REVOKED—已撤销（付款码支付）USERPAYING--用户支付中（付款码支付）PAYERROR--支付失败(其他原因，如银行返回失败)支付状态机请见下单API页面

	DeviceInfo         string `xml:"device_info,omitempty"`          // 微信支付分配的终端设备号
	OpenID             string `xml:"openid,omitempty"`               // 用户在商户appid下的唯一标识
	IsSubscribe        string `XML:"is_subscribe,omitempty"`         // 是否关注了公众号
	TradeType          string `xml:"trade_type,omitempty"`           // 交易类型 JSAPI，NATIVE，APP，MICROPAY，
	BankType           string `xml:"bank_type,omitempty"`            // 付款银行
	TotalFee           int    `xml:"total_fee,omitempty"`            // 标价金额
	SettlementTotalFee int    `xml:"settlement_total_fee,omitempty"` // 应结订单金额
	FeeType            string `xml:"fee_type,omitempty"`             // 货币种类 目前仅 CNY
	CashFee            int    `xml:"cash_fee,omitempty"`             // 现金支付金额
	CashFeeType        string `xml:"cash_fee_type,omitempty"`        // 货币类型，符合ISO 4217标准的三位字母代码，
	CouponFee          int    `xml:"coupon_fee,omitempty"`           // 代金券”金额<=订单金额，订单金额-“代金券”金额=现金支付金额，
	CouponCount        int    `xml:"coupon_count,omitempty"`         // 代金券 数量
	TransactionID      string `xml:"transaction_id,omitempty"`       // 微信订单号
	OutTradeNo         string `xml:"out_trade_no,omitempty"`         // 商户系统内部订单号，要求32个字符内，只能是数字、大小写字母_-|*@ ，且在同一个商户号下唯一。
	Attach             string `xml:"attach,omitempty"`               // 深圳分店	附加数据，原样返回
	TradeStateDesc     string `xml:"trade_state_desc,omitempty"`     // 对当前查询订单状态的描述和下一步操作的指引
	TimeEnd            string `xml:"time_end,omitempty"`             // 交易结束时间
}

// NewPay return an instance of Pay package
func NewPay(ctx *context.Context) *Pay {
	pay := Pay{Context: ctx}
	return &pay
}

// PrePayID will request wechat merchant api and request for a pre payment order id
func (pcf *Pay) PrePayID(p *UnifiedorderParams) (prePayID string, err error) {
	nonceStr := util.RandomStr(32)
	tradeType := "JSAPI"
	template := "appid=%s&body=%s&fee_type=%s&mch_id=%s&nonce_str=%s&notify_url=%s&openid=%s&out_trade_no=%s&spbill_create_ip=%s&total_fee=%s&trade_type=%s&key=%s"
	str := fmt.Sprintf(template, pcf.AppID, p.Body, p.FeeType, pcf.PayMchID, nonceStr, pcf.PayNotifyURL, p.OpenID, p.OutTradeNo, p.CreateIP, p.TotalFee, tradeType, pcf.PayKey)
	sign := util.MD5Sum(str)
	request := payRequest{
		AppID:          pcf.AppID,
		MchID:          pcf.PayMchID,
		NonceStr:       nonceStr,
		Sign:           sign,
		Body:           p.Body,
		OutTradeNo:     p.OutTradeNo,
		FeeType:        p.FeeType,
		TotalFee:       p.TotalFee,
		SpbillCreateIP: p.CreateIP,
		NotifyURL:      pcf.PayNotifyURL,
		TradeType:      tradeType,
		OpenID:         p.OpenID,
	}
	rawRet, err := util.PostXML(payGateway, request)
	if err != nil {
		return "", err
	}
	payRet := payResult{}
	err = xml.Unmarshal(rawRet, &payRet)
	if err != nil {
		return "", err
	}
	if payRet.ReturnCode == "SUCCESS" {
		//pay success
		if payRet.ResultCode == "SUCCESS" {
			return payRet.PrePayID, nil
		}
		return "", errors.New(payRet.ErrCode + payRet.ErrCodeDes)
	}
	return "", errors.New("[msg : xmlUnmarshalError] [rawReturn : " + string(rawRet) + "] [params : " + str + "] [sign : " + sign + "]")
}

// GetJSAPI 配置文件
func (pcf *Pay) GetJSAPI(p *UnifiedorderParams) (*JSAPIParams, error) {
	nonceStr := util.RandomStr(32)
	prePayID, err := pcf.PrePayID(p)
	if err != nil {
		return nil, err
	}

	pkg := "prepay_id=" + prePayID
	signType := "MD5"
	t := time.Now().Unix()
	timeStr := strconv.FormatInt(t, 10)
	template := "appId=%s&nonceStr=%s&package=%s&signType=%s&timeStamp=%s&key=%s"
	str := fmt.Sprintf(template, pcf.AppID, nonceStr, pkg, signType, timeStr, pcf.PayKey)
	sign := util.MD5Sum(str)
	var jp JSAPIParams
	jp.AppID = pcf.AppID
	jp.Timestamp = t
	jp.NonceStr = nonceStr
	jp.Package = pkg
	jp.SignType = signType
	jp.Sign = sign
	return &jp, nil
}

// OrderQuery 查询订单结果 自己判断 TradeState 是否成功
func (pcf *Pay) OrderQuery(outTradeNo string) (*OrderQueryResult, error) {
	nonceStr := util.RandomStr(32)
	template := "appid=%s&mch_id=%s&nonce_str=%s&out_trade_no=%s&key=%s"
	str := fmt.Sprintf(template, pcf.AppID, pcf.PayMchID, nonceStr, outTradeNo, pcf.PayKey)
	sign := util.MD5Sum(str)
	request := OrderQueryRequest{
		AppID:      pcf.AppID,
		MchID:      pcf.PayMchID,
		OutTradeNo: outTradeNo,
		NonceStr:   nonceStr,
		Sign:       sign,
	}
	rawRet, err := util.PostXML(orderQueryURI, request)
	if err != nil {
		// 失败了 使用备用接口再次查询
		rawRet, err := util.PostXML(orderQueryURI2, request)
		if err != nil {
			return nil, err
		}
		oqr, err := rawOrderQuery(rawRet, str, sign)
		if err != nil {
			return nil, err
		}
		return oqr, err

	}
	oqr, err := rawOrderQuery(rawRet, str, sign)
	if err != nil {
		return nil, err
	}
	return oqr, err
}

// OrderMchQuery 查询订单结果 自己判断 TradeState 是否成功
func (pcf *Pay) OrderMchQuery(TransactionID string) (*OrderQueryResult, error) {
	nonceStr := util.RandomStr(32)
	template := "appid=%s&mch_id=%s&nonce_str=%s&transaction_id=%s&key=%s"
	str := fmt.Sprintf(template, pcf.AppID, pcf.PayMchID, nonceStr, TransactionID, pcf.PayKey)
	sign := util.MD5Sum(str)
	request := OrderQueryRequest{
		AppID:         pcf.AppID,
		MchID:         pcf.PayMchID,
		TransactionID: TransactionID,
		NonceStr:      nonceStr,
		Sign:          sign,
	}
	rawRet, err := util.PostXML(orderQueryURI, request)
	if err != nil {
		// 失败了 使用备用接口再次查询
		rawRet, err := util.PostXML(orderQueryURI2, request)
		if err != nil {
			return nil, err
		}
		oqr, err := rawOrderQuery(rawRet, str, sign)
		if err != nil {
			return nil, err
		}
		return oqr, err

	}
	oqr, err := rawOrderQuery(rawRet, str, sign)
	if err != nil {
		return nil, err
	}
	return oqr, err
}
func rawOrderQuery(rawRet []byte, str, sign string) (*OrderQueryResult, error) {
	oqrRet := OrderQueryResult{}
	err := xml.Unmarshal(rawRet, &oqrRet)
	if err != nil {
		return nil, err
	}
	if oqrRet.ReturnCode == "SUCCESS" {
		if oqrRet.ResultCode == "SUCCESS" {
			// if oqrRet.TradeState == "SUCCESS" {
			return &oqrRet, nil
			// }
			// return nil, errors.New(oqrRet.TradeState)
		}
		return nil, errors.New(oqrRet.ErrCode + oqrRet.ErrCodeDes)
	}
	return nil, errors.New("[msg : xmlUnmarshalError] [rawReturn : " + string(rawRet) + "] [signstr : " + str + "] [sign : " + sign + "]")
}

// NotifyInfo 解码微信的通知信息 并验证权限
func (pcf *Pay) NotifyInfo(req *http.Request) (*OrderQueryResult, error) {
	oqrRet := OrderQueryResult{}
	reqByte, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	err = xml.Unmarshal(reqByte, &oqrRet)
	if err != nil {
		return nil, err
	}
	if oqrRet.ReturnCode == "SUCCESS" {
		if oqrRet.ResultCode == "SUCCESS" {
			return &oqrRet, nil
		}
		return nil, errors.New(oqrRet.ResultCode)
	}
	return nil, errors.New(oqrRet.ReturnCode)

}

// CheckSign 检查签名
func (pcf *Pay) CheckSign(or *OrderQueryResult, p *UnifiedorderParams) error {
	// appid=%s&mch_id=%s&result_code=%s&openid=%s&is_subscribe=%s&trade_type=%s&bank_type=%s&total_fee=%s&cash_fee=%s&transaction_id=%s&out_trade_no=%s&time_end=%s
	tmp := []string{
		"appid=" + pcf.AppID + "&",
		"mch_id=" + pcf.PayMchID + "&",
		"result_code=" + or.ResultCode + "&",
		"openid=" + p.OpenID + "&",
		"is_subscribe=" + or.IsSubscribe + "&",
		"trade_type=" + or.TradeType + "&",
		"bank_type=" + or.BankType + "&",
		"total_fee=" + p.TotalFee + "&",
		"cash_fee=" + strconv.Itoa(or.CashFee) + "&",
		"transaction_id=" + or.TransactionID + "&",
		"out_trade_no=" + p.OutTradeNo + "&",
		"time_end=" + or.TimeEnd + "&",
		"return_code=" + or.ReturnCode + "&",
		"return_msg=" + or.ReturnMsg + "&",
		"nonce_str=" + or.NonceStr + "&",
	}
	sort.Strings(tmp)
	var str string
	for _, v := range tmp {
		str += v
	}
	sign := util.MD5Sum(str)
	if sign == or.Sign {
		return nil
	}
	return errors.New("签名错误")
}
