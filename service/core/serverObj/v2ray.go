package serverObj

import (
	"encoding/base64"
	"fmt"
	"github.com/iopq/xraya/common/ntp"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/tidwall/gjson"
	"github.com/iopq/xraya/common"
	"github.com/iopq/xraya/core/coreObj"
	"github.com/iopq/xraya/core/v2ray/service"
	"github.com/iopq/xraya/pkg/util/log"
)

func init() {
	FromLinkRegister("vmess", NewV2Ray)
	FromLinkRegister("vless", NewV2Ray)
	EmptyRegister("vmess", func() (ServerObj, error) {
		return new(V2Ray), nil
	})
	EmptyRegister("vless", func() (ServerObj, error) {
		return new(V2Ray), nil
	})
}

type V2Ray struct {
	Ps            string `json:"ps"`
	Add           string `json:"add"`
	Port          string `json:"port"`
	ID            string `json:"id"`
	Aid           string `json:"aid"`
	Net           string `json:"net"`
	Type          string `json:"type"`
	Host          string `json:"host"`
	Path          string `json:"path"`
	TLS           string `json:"tls"`
	Flow          string `json:"flow,omitempty"`
	Alpn          string `json:"alpn,omitempty"`
	AllowInsecure bool   `json:"allowInsecure"`
	V             string `json:"v"`
	Protocol      string `json:"protocol"`
	Fingerprint   string `json:"fingerprint"`
	PublicKey     string `json:"publicKey"`
	ShortId       string `json:"shortId"`
	SpiderX       string `json:"spiderX"`
}

func NewV2Ray(link string) (ServerObj, error) {
	if strings.HasPrefix(link, "vmess://") {
		return ParseVmessURL(link)
	} else if strings.HasPrefix(link, "vless://") {
		return ParseVlessURL(link)
	}
	return nil, InvalidParameterErr
}

func ParseVlessURL(vless string) (data *V2Ray, err error) {
	u, err := url.Parse(vless)
	if err != nil {
		return nil, err
	}
	data = &V2Ray{
		Ps:            u.Fragment,
		Add:           u.Hostname(),
		Port:          u.Port(),
		ID:            u.User.String(),
		Net:           u.Query().Get("type"),
		Type:          u.Query().Get("headerType"),
		Host:          u.Query().Get("sni"),
		Path:          u.Query().Get("path"),
		TLS:           u.Query().Get("security"),
		Flow:          u.Query().Get("flow"),
		Alpn:          u.Query().Get("alpn"),
		AllowInsecure: u.Query().Get("allowInsecure") == "true",
		Protocol:      "vless",
		Fingerprint:   u.Query().Get("fingerprint"),
		PublicKey:     u.Query().Get("publicKey"),
		ShortId:       u.Query().Get("shortId"),
		SpiderX:       u.Query().Get("spiderX"),
	}
	if data.Net == "" {
		data.Net = "tcp"
	}
	if data.Net == "grpc" {
		data.Path = u.Query().Get("serviceName")
	}
	if data.Type == "" {
		data.Type = "none"
	}
	if data.Host == "" {
		data.Host = u.Query().Get("host")
	}
	if data.TLS == "" {
		data.TLS = "none"
	}
	if data.Flow == "" {
		data.Flow = "xtls-rprx-direct"
	}
	if data.Type == "mkcp" || data.Type == "kcp" {
		data.Path = u.Query().Get("seed")
	}
	return data, nil
}

func ParseVmessURL(vmess string) (data *V2Ray, err error) {
	var info V2Ray
	// perform base64 decoding and unmarshal to VmessInfo
	raw, err := common.Base64StdDecode(vmess[8:])
	if err != nil {
		raw, err = common.Base64URLDecode(vmess[8:])
	}
	if err != nil {
		// not in json format, try to resolve as vmess://BASE64(Security:ID@Add:Port)?remarks=Ps&obfsParam=Host&Path=Path&obfs=Net&tls=TLS
		var u *url.URL
		u, err = url.Parse(vmess)
		if err != nil {
			return
		}
		re := regexp.MustCompile(`.*:(.+)@(.+):(\d+)`)
		s := strings.Split(vmess[8:], "?")[0]
		s, err = common.Base64StdDecode(s)
		if err != nil {
			s, err = common.Base64URLDecode(s)
		}
		subMatch := re.FindStringSubmatch(s)
		if subMatch == nil {
			err = fmt.Errorf("unrecognized vmess address")
			return
		}
		q := u.Query()
		ps := q.Get("remarks")
		if ps == "" {
			ps = q.Get("remark")
		}
		obfs := q.Get("obfs")
		obfsParam := q.Get("obfsParam")
		path := q.Get("path")
		if obfs == "kcp" || obfs == "mkcp" {
			m := make(map[string]string)
			//cater to v2rayN definition
			_ = jsoniter.Unmarshal([]byte(obfsParam), &m)
			path = m["seed"]
			obfsParam = ""
		}
		aid := q.Get("alterId")
		if aid == "" {
			aid = q.Get("aid")
		}
		info = V2Ray{
			ID:            subMatch[1],
			Add:           subMatch[2],
			Port:          subMatch[3],
			Ps:            ps,
			Host:          obfsParam,
			Path:          path,
			Net:           obfs,
			Aid:           aid,
			TLS:           map[string]string{"1": "tls"}[q.Get("tls")],
			AllowInsecure: false,
		}
		if info.Net == "websocket" {
			info.Net = "ws"
		}
	} else {
		err = jsoniter.Unmarshal([]byte(raw), &info)
		if err != nil {
			return
		}
		if info.Host == "" {
			sni := gjson.Get(raw, "sni")
			if sni.Exists() {
				info.Host = sni.String()
			}
		}
	}
	// correct the wrong vmess as much as possible
	if strings.HasPrefix(info.Host, "/") && info.Path == "" {
		info.Path = info.Host
		info.Host = ""
	}
	if info.Aid == "" {
		info.Aid = "0"
	}
	info.Protocol = "vmess"
	return &info, nil
}

func (v *V2Ray) Configuration(info PriorInfo) (c Configuration, err error) {
	core := coreObj.OutboundObject{
		Tag:      info.Tag,
		Protocol: v.Protocol,
	}
	port, _ := strconv.Atoi(v.Port)
	switch strings.ToLower(v.Protocol) {
	case "vmess", "vless":
		id := v.ID
		if l := len([]byte(id)); l < 32 || l > 36 {
			id = common.StringToUUID5(id)
		}
		switch strings.ToLower(v.Protocol) {
		case "vmess":
			if ok, t, err := ntp.IsDatetimeSynced(); err == nil && !ok {
				return Configuration{}, fmt.Errorf("please sync datetime first. Your datetime is %v, and the correct datetime is %v", time.Now().Local().Format(ntp.DisplayFormat), t.Local().Format(ntp.DisplayFormat))
			}
			core.Settings.Vnext = []coreObj.Vnext{
				{
					Address: v.Add,
					Port:    port,
					Users: []coreObj.User{
						{
							ID:       id,
							AlterID:  0,
							Security: "auto",
						},
					},
				},
			}
		case "vless":
			core.Settings.Vnext = []coreObj.Vnext{
				{
					Address: v.Add,
					Port:    port,
					Users: []coreObj.User{
						{
							ID: id,
							//AlterID:    0, // keep AEAD on
							Encryption: "none",
						},
					},
				},
			}
		}
		core.StreamSettings = &coreObj.StreamSettings{
			Network: v.Net,
		}
		// 根据传输协议(network)修改streamSettings
		//TODO: QUIC
		switch strings.ToLower(v.Net) {
		case "grpc":
			if err := service.CheckGrpcSupported(); err != nil {
				return Configuration{}, err
			}
			if v.Path == "" {
				v.Path = "GunService"
			}
			core.StreamSettings.GrpcSettings = &coreObj.GrpcSettings{ServiceName: v.Path}
		case "ws":
			core.StreamSettings.WsSettings = &coreObj.WsSettings{
				Path: v.Path,
				Headers: coreObj.Headers{
					Host: v.Host,
				},
			}
		case "mkcp", "kcp":
			core.StreamSettings.KcpSettings = &coreObj.KcpSettings{
				Mtu:              1350,
				Tti:              50,
				UplinkCapacity:   12,
				DownlinkCapacity: 100,
				Congestion:       false,
				ReadBufferSize:   2,
				WriteBufferSize:  2,
				Header: coreObj.KcpHeader{
					Type: v.Type,
				},
				Seed: v.Path,
			}
		case "tcp":
			if strings.ToLower(v.Type) == "http" {
				tcpSetting := coreObj.TCPSettings{
					ConnectionReuse: true,
					Header: coreObj.TCPHeader{
						Type: "http",
						Request: coreObj.HTTPRequest{
							Version: "1.1",
							Method:  "GET",
							Path:    strings.Split(v.Path, ","),
							Headers: coreObj.HTTPReqHeader{
								Host: strings.Split(v.Host, ","),
								UserAgent: []string{
									"Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/55.0.2883.75 Safari/537.36",
									"Mozilla/5.0 (iPhone; CPU iPhone OS 10_0_2 like Mac OS X) AppleWebKit/601.1 (KHTML, like Gecko) CriOS/53.0.2785.109 Mobile/14A456 Safari/601.1.46",
								},
								AcceptEncoding: []string{"gzip, deflate"},
								Connection:     []string{"keep-alive"},
								Pragma:         "no-cache",
							},
						},
						Response: coreObj.HTTPResponse{
							Version: "1.1",
							Status:  "200",
							Reason:  "OK",
							Headers: coreObj.HTTPRespHeader{
								ContentType:      []string{"application/octet-stream", "video/mpeg"},
								TransferEncoding: []string{"chunked"},
								Connection:       []string{"keep-alive"},
								Pragma:           "no-cache",
							},
						},
					},
				}
				tcpSetting.Header.Request.Headers.Host = strings.Split(v.Host, ",")
				if v.Path != "" {
					tcpSetting.Header.Request.Path = strings.Split(v.Path, ",")
					for i := range tcpSetting.Header.Request.Path {
						if !strings.HasPrefix(tcpSetting.Header.Request.Path[i], "/") {
							tcpSetting.Header.Request.Path[i] = "/" + tcpSetting.Header.Request.Path[i]
						}
					}
				}
				core.StreamSettings.TCPSettings = &tcpSetting
			}
		case "h2", "http":
			if v.Host != "" {
				core.StreamSettings.HTTPSettings = &coreObj.HttpSettings{
					Path: v.Path,
					Host: strings.Split(v.Host, ","),
				}
			} else {
				core.StreamSettings.HTTPSettings = &coreObj.HttpSettings{
					Path: v.Path,
				}
			}
		}
		if strings.ToLower(v.TLS) == "tls" {
			core.StreamSettings.Security = "tls"
			core.StreamSettings.TLSSettings = &coreObj.TLSSettings{}
			if v.AllowInsecure {
				core.StreamSettings.TLSSettings.AllowInsecure = true
			}
			// SNI
			if v.Host != "" {
				core.StreamSettings.TLSSettings.ServerName = v.Host
			}

			// Alpn
			if v.Alpn != "" {
				alpn := strings.Split(v.Alpn, ",")
				for i := range alpn {
					alpn[i] = strings.TrimSpace(alpn[i])
				}
				core.StreamSettings.TLSSettings.Alpn = alpn
			}
			vnext := core.Settings.Vnext.([]coreObj.Vnext)
			vnext[0].Users[0].Flow = v.Flow
		} else if strings.ToLower(v.TLS) == "xtls" {
			core.StreamSettings.Security = "xtls"
			core.StreamSettings.XTLSSettings = &coreObj.TLSSettings{}
			// SNI
			if v.Host != "" {
				core.StreamSettings.XTLSSettings.ServerName = v.Host
			}
			if v.AllowInsecure {
				core.StreamSettings.XTLSSettings.AllowInsecure = true
			}
			if v.Flow == "" {
				v.Flow = "xtls-rprx-origin"
			}
			if v.Alpn != "" {
				alpn := strings.Split(v.Alpn, ",")
				for i := range alpn {
					alpn[i] = strings.TrimSpace(alpn[i])
				}
				core.StreamSettings.XTLSSettings.Alpn = alpn
			}
			vnext := core.Settings.Vnext.([]coreObj.Vnext)
			vnext[0].Users[0].Flow = v.Flow
			core.Settings.Vnext = vnext
		} else if strings.ToLower(v.TLS) == "reality" {
			core.StreamSettings.Security = "reality"
			core.StreamSettings.RealitySettings = &coreObj.TLSSettings{}
			// SNI
			if v.Host != "" {
				core.StreamSettings.RealitySettings.ServerName = v.Host
			}
			if v.AllowInsecure {
				core.StreamSettings.RealitySettings.AllowInsecure = true
			}
			if v.Flow == "" {
				v.Flow = "xtls-rprx-origin"
			}
			if v.Alpn != "" {
				alpn := strings.Split(v.Alpn, ",")
				for i := range alpn {
					alpn[i] = strings.TrimSpace(alpn[i])
				}
				core.StreamSettings.TLSSettings.Alpn = alpn
			}
			if v.Fingerprint != "" {
			        core.StreamSettings.RealitySettings.Fingerprint = v.Fingerprint
			}
			if v.PublicKey != "" {
			        core.StreamSettings.RealitySettings.PublicKey = v.PublicKey
			}
			core.StreamSettings.RealitySettings.ShortId = v.ShortId
			core.StreamSettings.RealitySettings.SpiderX = v.SpiderX
			
			vnext := core.Settings.Vnext.([]coreObj.Vnext)
			vnext[0].Users[0].Flow = v.Flow
			core.Settings.Vnext = vnext
		}
	}
	return Configuration{
		CoreOutbound: core,
		PluginChain:  "",
		UDPSupport:   true,
	}, nil
}

func (v *V2Ray) ExportToURL() string {
	switch v.Protocol {
	case "vless":
		// https://github.com/XTLS/Xray-core/issues/91
		var query = make(url.Values)
		setValue(&query, "type", v.Net)
		setValue(&query, "security", v.TLS)
		switch v.Net {
		case "websocket", "ws", "http", "h2":
			setValue(&query, "path", v.Path)
			setValue(&query, "host", v.Host)
		case "mkcp", "kcp":
			setValue(&query, "headerType", v.Type)
			setValue(&query, "seed", v.Path)
		case "tcp":
			setValue(&query, "headerType", v.Type)
			setValue(&query, "host", v.Host)
			setValue(&query, "path", v.Path)
		case "grpc":
			setValue(&query, "serviceName", v.Path)
		}
		//TODO: QUIC
		if v.TLS != "none" {
			setValue(&query, "sni", v.Host) // FIXME: it may be different from ws's host
			setValue(&query, "alpn", v.Alpn)
			setValue(&query, "allowInsecure", strconv.FormatBool(v.AllowInsecure))
		}
		if v.TLS == "xtls" || v.TLS == "tls" || v.TLS == "reality" {
			setValue(&query, "flow", v.Flow)
		}
		
		if v.TLS == "reality" {
		       setValue(&query, "fingerprint", v.FingerPrint)
		       setValue(&query, "publicKey", v.PublicKey)
		       setValue(&query, "shortId", v.ShortId)
		       setValue(&query, "spiderX", v.SpiderX)
		}

		U := url.URL{
			Scheme:   "vless",
			User:     url.User(v.ID),
			Host:     net.JoinHostPort(v.Add, v.Port),
			RawQuery: query.Encode(),
			Fragment: v.Ps,
		}
		return U.String()
	case "vmess":
		v.V = "2"
		b, _ := jsoniter.Marshal(v)
		return "vmess://" + strings.TrimSuffix(base64.StdEncoding.EncodeToString(b), "=")
	}
	log.Warn("unexpected protocol: %v", v.Protocol)
	return ""
}

func (v *V2Ray) NeedPluginPort() bool {
	return false
}

func (v *V2Ray) ProtoToShow() string {
	if v.TLS != "" && v.TLS != "none" {
		return fmt.Sprintf("%v(%v+%v)", v.Protocol, v.Net, v.TLS)
	}
	return fmt.Sprintf("%v(%v)", v.Protocol, v.Net)
}

func (v *V2Ray) GetProtocol() string {
	return v.Protocol
}

func (v *V2Ray) GetHostname() string {
	return v.Add
}

func (v *V2Ray) GetPort() int {
	p, _ := strconv.Atoi(v.Port)
	return p
}

func (v *V2Ray) GetName() string {
	return v.Ps
}

func (v *V2Ray) SetName(name string) {
	v.Ps = name
}
