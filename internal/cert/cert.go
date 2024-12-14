// Пакет cert предоставляет утилиты для генерации и управления TLS-сертификатами,
// используемыми для обеспечения безопасного соединения в приложении.
package cert

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"github.com/sol1corejz/go-url-shortener/internal/logger"
	"go.uber.org/zap"
	"math/big"
	"net"
	"os"
	"time"
)

const (
	// CertificateFilePath задаёт путь к файлу для хранения сгенерированного TLS-сертификата.
	CertificateFilePath = "server.crt"
	// KeyFilePath задаёт путь к файлу для хранения сгенерированного приватного ключа.
	KeyFilePath = "server.key"
)

// GenerateCert генерирует самоподписанный сертификат и приватный ключ в формате PEM.
// Возвращает срезы байт для сертификата и ключа.
func GenerateCert() ([]byte, []byte) {
	// Создаём шаблон для нового сертификата.
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(1658), // Уникальный номер сертификата.
		Subject: pkix.Name{
			Organization: []string{"sol1.kek"},
			Country:      []string{"RU"},
		},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback}, // Разрешённые IP-адреса.
		NotBefore:    time.Now(),                                         // Начало срока действия.
		NotAfter:     time.Now().AddDate(10, 0, 0),                       // Срок действия — 10 лет.
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	// Генерируем новый RSA-приватный ключ длиной 4096 бит.
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		logger.Log.Error("Ошибка генерации приватного ключа:", zap.Error(err))
	}

	// Создаём сертификат на основе шаблона.
	certBytes, err := x509.CreateCertificate(rand.Reader, cert, cert, &privateKey.PublicKey, privateKey)
	if err != nil {
		logger.Log.Error("Ошибка создания сертификата:", zap.Error(err))
	}

	// Кодируем сертификат в формате PEM.
	var certPEM bytes.Buffer
	pem.Encode(&certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	// Кодируем приватный ключ в формате PEM.
	var privateKeyPEM bytes.Buffer
	pem.Encode(&privateKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	return certPEM.Bytes(), privateKeyPEM.Bytes()
}

// CertExists проверяет, существуют ли файлы с сертификатом и приватным ключом.
// Возвращает true, если оба файла существуют.
func CertExists() bool {
	_, certErr := os.Stat(CertificateFilePath)
	_, keyErr := os.Stat(KeyFilePath)
	return certErr == nil && keyErr == nil
}

// SaveCert сохраняет переданные сертификат и приватный ключ в файлы.
// Возвращает ошибку, если запись в файл не удалась.
func SaveCert(certPEM, keyPEM []byte) error {
	if err := os.WriteFile(CertificateFilePath, certPEM, 0600); err != nil {
		return err
	}
	if err := os.WriteFile(KeyFilePath, keyPEM, 0600); err != nil {
		return err
	}
	return nil
}
