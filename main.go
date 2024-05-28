package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/elazarl/goproxy"
	"github.com/joho/godotenv"
)

var rejects = []string{}

func main() {
	// .envファイルを読み込む
	godotenv.Load(".env")

	REDIRECT_TO := os.Getenv("REDIRECT_TO")
	fmt.Println("Redirect to : ", REDIRECT_TO)

	// CAの設定を行う
	goproxyCa, err := tls.X509KeyPair(caCert, caKey)
	if err != nil {
		log.Fatal(err)
	}
	if goproxyCa.Leaf, err = x509.ParseCertificate(goproxyCa.Certificate[0]); err != nil {
		log.Fatal(err)
	}
	goproxy.GoproxyCa = goproxyCa
	goproxy.OkConnect = &goproxy.ConnectAction{Action: goproxy.ConnectAccept, TLSConfig: goproxy.TLSConfigFromCA(&goproxyCa)}
	goproxy.MitmConnect = &goproxy.ConnectAction{Action: goproxy.ConnectMitm, TLSConfig: goproxy.TLSConfigFromCA(&goproxyCa)}
	goproxy.HTTPMitmConnect = &goproxy.ConnectAction{Action: goproxy.ConnectHTTPMitm, TLSConfig: goproxy.TLSConfigFromCA(&goproxyCa)}
	goproxy.RejectConnect = &goproxy.ConnectAction{Action: goproxy.ConnectReject, TLSConfig: goproxy.TLSConfigFromCA(&goproxyCa)}

	// プロキシサーバーを起動
	proxy := goproxy.NewProxyHttpServer()

	proxy.ConnectDial = nil
	proxy.Verbose = false
	proxy.CertStore = NewCertStorage()
	proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)

	// ログディレクトリの作成
	err = os.Mkdir("./logs", 0770)
	if err != nil && !os.IsExist(err) {
		log.Fatal(err)
	}
	// ログを書き込むためのファイルを作成
	f, err := os.Create("./logs/latest.log")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	defer os.Rename("./logs/latest.log", fmt.Sprintf("./logs/%d.log", time.Now().Unix()))

	// ログ収集サーバーのアドレスを取得
	LC_ADDRESS := os.Getenv("LC_ADDRESS")

	if LC_ADDRESS == "" {
		log.Fatal("ログ収集サーバーが指定されていません")
	}

	serverAddr, err := net.ResolveTCPAddr("tcp", LC_ADDRESS)
	if err != nil {
		log.Fatal(err)
	}
	conn, err := net.DialTCP("tcp", nil, serverAddr)
	if err != nil {
		log.Fatal(err)
	}

	defer conn.Close()

	go handleConnection(conn)

	// プロキシ経由でリクエストがあった場合に処理を行う
	proxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		isReject := false
		for _, suffix := range rejects {
			if endsWith(req.Host, suffix) {
				isReject = true
				break
			}
		}

		// URLのデコード
		decoded_url, _ := url.QueryUnescape(req.URL.String())

		// ログを記録するためにデータを取得する
		data := make([]string, 3)
		data[0] = req.RemoteAddr               // from
		data[1] = decoded_url                  // to
		data[2] = strconv.FormatBool(isReject) // is reject

		// 収集サーバーにデータを送信
		conn.SetWriteDeadline(time.Now().Add(time.Minute))
		conn.Write([]byte(strings.Join(data, "\t")))

		if isReject {
			newUrl, _ := url.Parse(REDIRECT_TO)
			req.URL = newUrl

			// 代わりに宛先の内容を取得する
			client := &http.Client{}

			request, _ := http.NewRequest("GET", REDIRECT_TO, nil)
			for _, v := range req.Cookies() {
				request.AddCookie(v)
			}
			request.Header = req.Header.Clone()
			response, _ := client.Do(request)

			return req, response
		}

		return req, nil
	})

	// エラーがあった場合は出力して終了
	log.Fatal(http.ListenAndServe(":8000", proxy))

}

func handleConnection(conn net.Conn) {
	// データを受け取るためのバッファ
	buff := make([]byte, 1024)
	for {
		messageLength, err := conn.Read(buff)
		if err != nil {
			log.Fatal(err)
		}
		message := string(buff[:messageLength])

		data := strings.Split(message, "\t")
		if len(data) == 0 {
			continue
		}
		if data[0] == "ADD" && len(data) == 2 && data[1] != "" {
			fmt.Println("New reject host : ", data[1])
			conn.Write([]byte("Accept"))
			rejects = append(rejects, data[1])
		}
		if data[0] == "REMOVE" && len(data) == 2 {
			newRej := rejects[0:0]
			for _, v := range rejects {
				if v != data[1] {
					newRej = append(newRej, v)
				}
			}
			rejects = newRej
		}
	}
}

func endsWith(text, suffix string) bool {
	if len(text) < len(suffix) {
		return false
	}
	textLen, suffixLen := len(text), len(suffix)

	for i := 1; i <= suffixLen; i++ {
		if text[textLen-i] != suffix[suffixLen-i] {
			return false
		}
	}
	return true
}
