package proxy

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"github.com/tdwu/fast_go/fast_base"
	"io"
	"math/big"
	"net"
	"net/http"
	"time"
)

var httpProxyServer *http.Server

func StopProxy() {
	if httpProxyServer != nil {
		fast_base.Logger.Info("关闭代理服务....")
		httpProxyServer.Shutdown(context.Background())
		httpProxyServer = nil
		fast_base.Logger.Info("关闭代理服务done")
	}
}

func StartProxy() {
	// 先尝试关闭
	StopProxy()

	port := fast_base.ConfigAll.GetString("ProxyPort")
	if port == "" {
		// 未配置，则不启动代理服务
		return
	}

	tlsCfg := &tls.Config{InsecureSkipVerify: false} // 设置为校验目标服务器的证书
	ssl := fast_base.ConfigAll.GetString("ProxyUseSSL")
	if ssl == "1" {
		cert, err := genCertificate()
		if err != nil {
			fast_base.Logger.Error("获取代理服务器的证书失败," + err.Error())
		}
		tlsCfg = &tls.Config{Certificates: []tls.Certificate{cert}}
	}

	httpProxyServer = &http.Server{
		Addr:      ":" + port,
		TLSConfig: tlsCfg,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodConnect {
				fast_base.Logger.Info("代理(https)：" + r.URL.String())
				handleHttpsRequest(w, r)
			} else {
				fast_base.Logger.Info("代理(http)：" + r.URL.String())
				handleHttpRequest(w, r)
			}
		}),
	}

	go func() {
		fast_base.Logger.Info("启动代理服务(正向), 使用端口:" + port)
		err := httpProxyServer.ListenAndServe()
		if err != nil {
			fast_base.Logger.Error("Http proxy server start failed." + err.Error())
		}
		fast_base.Logger.Info("代理服务结束")
	}()

}

func handleHttpsRequest(w http.ResponseWriter, r *http.Request) {
	destConn, err := net.DialTimeout("tcp", r.Host, 60*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}
	go transfer(destConn, clientConn)
	go transfer(clientConn, destConn)

}

func handleHttpRequest(w http.ResponseWriter, r *http.Request) {
	resp, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)

}

func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	io.Copy(destination, source)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func genCertificate() (cert tls.Certificate, err error) {
	rawCert, rawKey, err := generateKeyPair()
	if err != nil {
		return
	}
	return tls.X509KeyPair(rawCert, rawKey)

}

func generateKeyPair() (rawCert, rawKey []byte, err error) {
	// Create private key and self-signed certificate
	// Adapted from https://golang.org/src/crypto/tls/generate_cert.go

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return
	}
	validFor := time.Hour * 24 * 365 * 10 // ten years
	notBefore := time.Now()
	notAfter := notBefore.Add(validFor)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Zarten"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return
	}

	rawCert = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	rawKey = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	return
}
