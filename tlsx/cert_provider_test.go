package tlsx

import (
	"bytes"
	"context"
	"crypto/tls"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCertProvider(t *testing.T) {
	k1priv := `-----BEGIN PRIVATE KEY-----
MIIBVAIBADANBgkqhkiG9w0BAQEFAASCAT4wggE6AgEAAkEAu+t3oH41zzzesJWw
LDj36bdf/Q7Aad6jsL1LCSv2FL6557v2dT6BTV+VzWtZC8uZUVwdBTHvEujc8xh8
Dn9OYQIDAQABAkEAop14v6F33wXFjvl5oksJ/W152vpQ90x6Sg8ER8OLBxcmhEIq
aDswT/54DpmCeLhTIK81Mu4bZIa+vgvxyzUAAQIhAPaNhg0B6/26K3aLsmg1lZHI
XkOr6XssjiSsaeyvZ3IBAiEAwx7NhG7/7DbMOhC2AsZF3YgvxAhppcJN/lkyAm3V
HGECIDoX6qgR9dsZDLin/eeUCKQLBDsJvL/rJar6fRLp2YQBAiA07r9MRRyShU8k
FXJ7EDTV42Mp6CpY+HxWGvZxKECfIQIgVhzRaKvw6iHtcmZMNL/QORBN01GGRFxy
JSVeA4IKnLY=
-----END PRIVATE KEY-----`
	k1crt := `-----BEGIN CERTIFICATE-----
MIIB4TCCAYugAwIBAgIUO9GxImIjDW94nod81+3oixgbDTQwDQYJKoZIhvcNAQEL
BQAwRTELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoM
GEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDAeFw0yMjA2MDgwNzQxMzVaFw0yMzA2
MDgwNzQxMzVaMEUxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEw
HwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwXDANBgkqhkiG9w0BAQEF
AANLADBIAkEAu+t3oH41zzzesJWwLDj36bdf/Q7Aad6jsL1LCSv2FL6557v2dT6B
TV+VzWtZC8uZUVwdBTHvEujc8xh8Dn9OYQIDAQABo1MwUTAdBgNVHQ4EFgQUVXSc
VIG7royHzQxJRYvLSQr+04MwHwYDVR0jBBgwFoAUVXScVIG7royHzQxJRYvLSQr+
04MwDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAANBAHEQjXMZGbzfeYs8
GUJIoNOf7oVKQF1fLnBvMr3dYZM1NaAORIMCphbANylV54q+mvoQodhYrI/rWxjO
gRjJAsU=
-----END CERTIFICATE-----`
	k2priv := `-----BEGIN PRIVATE KEY-----
MIIBVAIBADANBgkqhkiG9w0BAQEFAASCAT4wggE6AgEAAkEAvaiL9hSWw/a7CmHD
AXBfFn0Tx/Pm4YuwjJY0T/j+agTkKATVwN1JM+FRnVjtSXurztn4CMV8sM0ynTpU
IxenWwIDAQABAkB5ow+g07OeGy/6iJi444kYsz9sjlEVdrHUeME0SU1iUIOkboPH
/ZXKieFTKltEygldeTccRC+eyD+qwQLYpY8BAiEA+4rKr4qYWOQD7qSgL8bv18Ec
+vGvPZ+JH2yqEA/h7qkCIQDBBP9vQnExv6Q1ewm6+0D/kWB7KBJfmPcZaH7WSbr8
YwIgW05lBlVLubCC0ORHFTCkLO/3QgvqrXa0goiiLpRlUYkCIQCDTwsWfXTUCzOC
znkIIvVM53FjVxdowX8YYeYnkXELUQIgTbecX8gqmq6bAjzNOTSq095q5qRVxcX/
wARh6Zgnizk=
-----END PRIVATE KEY-----`
	k2crt := `-----BEGIN CERTIFICATE-----
MIIB4TCCAYugAwIBAgIUXIs5A6R08x4nREVre0cngZJP+RQwDQYJKoZIhvcNAQEL
BQAwRTELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoM
GEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDAeFw0yMjA2MDgwNzQzMjNaFw0yMzA2
MDgwNzQzMjNaMEUxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEw
HwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwXDANBgkqhkiG9w0BAQEF
AANLADBIAkEAvaiL9hSWw/a7CmHDAXBfFn0Tx/Pm4YuwjJY0T/j+agTkKATVwN1J
M+FRnVjtSXurztn4CMV8sM0ynTpUIxenWwIDAQABo1MwUTAdBgNVHQ4EFgQU92sq
fqVvB3rLQR3/v9fmURv438YwHwYDVR0jBBgwFoAU92sqfqVvB3rLQR3/v9fmURv4
38YwDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAANBAHaF1we2MGmtd60u
S8zLBpme53pJyY2r48Ol6xUSVLIoHOZ2V1TH0iHi4KnTMVyJwryyGZyhleAP7QA1
d/hJs+A=
-----END CERTIFICATE-----`

	certFixture := `LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUVFRENDQXZpZ0F3SUJBZ0lKQU5mK0lUMU1HaHhCTUEwR0NTcUdTSWI` +
		`zRFFFQkN3VUFNSUdaTVFzd0NRWUQKVlFRR0V3SlZVekVMTUFrR0ExVUVDQXdDUTBFeEVqQVFCZ05WQkFjTUNWQmhiRzhnUVd4MGJ6RWlNQ0FHQ` +
		`TFVRQpDZ3daVDI1bFEyOXVZMlZ5YmlCYmRHVnpkQ0J3ZFhKd2IzTmxYVEVjTUJvR0ExVUVBd3dUYjI1bFkyOXVZMlZ5CmJpMTBaWE4wTG1OdmJ` +
		`URW5NQ1VHQ1NxR1NJYjNEUUVKQVJZWVpuSmxaR1Z5YVdOQVkyOXVaV052Ym1ObGNtNHUKWTI5dE1CNFhEVEU0TURnd016RTJNakUwT0ZvWERUR` +
		`TVNVEl4TmpFMk1qRTBPRm93Z1lReEN6QUpCZ05WQkFZVApBbFZUTVFzd0NRWURWUVFJREFKRFFURVNNQkFHQTFVRUJ3d0pVR0ZzYnlCQmJIUnZ` +
		`NU0l3SUFZRFZRUUxEQmxQCmJtVkRiMjVqWlhKdUlGdDBaWE4wSUhCMWNuQnZjMlZkTVRBd0xnWURWUVFERENkaGNHa3RjMlZ5ZG1salpTMXcKY` +
		`205NGFXVmtMbTl1WldOdmJtTmxjbTR0ZEdWemRDNWpiMjB3Z2dFaU1BMEdDU3FHU0liM0RRRUJBUVVBQTRJQgpEd0F3Z2dFS0FvSUJBUURXVzF` +
		`KQnZweC9vZkYwei80QnkrYmdBcCtoYnlxblVsQ2FnYmlneE9QTHY3aUg4TSt1CjNENkRlSVkzQzdkV0thTjRnYXZHd1MvN3I0UWxXSWdvK09NR` +
		`HQ1M25OZDVvakwvNWY5R1E0ZGRObW53b25EeEYKVThrd1lMWURMTkJIQzJqMzFBNVNueHo0S1NkVE03Rmc0OFBJeTNBaWFGMkhEcURZVlJpWkV` +
		`ackl4U3JTSmFKZgp1WGVCSUVBcFBpUG1IOURObGw2VVo3ODZvZitJWWVLV2VuY0MvbGpPaGlJSnJWL3NEZTc2QVFjdXY5T29XaUdiCklGVFMyW` +
		`ExSRGF0YzByQXhWdlFiTnMzeWlFYjh3UzBaR0F4cTBuZk9pMGZkYVBIODdFc25MdkpqWk5PcXIvTVMKSW5BYmN2ZmlwckxxaEdLQTVIN2hKVGZ` +
		`EcFJ6WWxBcm5maTJMQWdNQkFBR2piakJzTUFrR0ExVWRFd1FDTUFBdwpDd1lEVlIwUEJBUURBZ1hnTUZJR0ExVWRFUVJMTUVtQ0htOWhkR2hyW` +
		`ldWd1pYSXViMjVsWTI5dVkyVnliaTEwClpYTjBMbU52YllJbllYQnBMWE5sY25acFkyVXRjSEp2ZUdsbFpDNXZibVZqYjI1alpYSnVMWFJsYzN` +
		`RdVkyOXQKTUEwR0NTcUdTSWIzRFFFQkN3VUFBNElCQVFCMVBibCtSbW50RW9jbHlqWXpzeWtLb2lYczNwYTgzQ2dEWjZwQwpncnY0TFF4U29FZ` +
		`kowNGY4YkQ0SUlZRkdDWmZWTkcwVnBFWHJObGs2VWJzVmRUQUJ0cUNndUpUV3dER1VBaDZYCjNiRmhyWm5QZXhzLy9Rd2dEQWRxSWYwRWd3Y0R` +
		`VRzc2R0lkZms3MGUxWnV4Y2h4ZDhVQkNwQUlkZVUwOHZWa3kKNFBXdjJLNGFENEZqQ2hLeENONWtoTjUwRk1QY2FJK3hWZ2Q0N3RQaFZOOWxRa` +
		`W9HRENoc1Q1dkFSazdiYS9jZQowUTlOV2RpTWZMRWdMZGNCb2JaS0Z0RnJsS3R5ek9nRGpMdlh2TFFzL3MybWVyU0k5Zmt3b09CRVArN2o3Wm5` +
		`zCkFqeTlNZmh3cWJUcFc3S3BDU0ZhMFZULzJ1OTVaUmNQdnJYbGRLUnlnQjRXdUFScgotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==`
	keyFixture := `LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFcEFJQkFBS0NBUUVBMWx0U1FiNmNmNkh4ZE0vK0Fjdm00QUtm` +
		`b1c4cXAxSlFtb0c0b01Uank3KzRoL0RQCnJ0dytnM2lHTnd1M1ZpbWplSUdyeHNFdis2K0VKVmlJS1BqakE3ZWQ1elhlYUl5LytYL1JrT0hYVF` +
		`pwOEtKdzgKUlZQSk1HQzJBeXpRUnd0bzk5UU9VcDhjK0NrblV6T3hZT1BEeU10d0ltaGRodzZnMkZVWW1SR2F5TVVxMGlXaQpYN2wzZ1NCQUtU` +
		`NGo1aC9RelpaZWxHZS9PcUgvaUdIaWxucDNBdjVZem9ZaUNhMWY3QTN1K2dFSExyL1RxRm9oCm15QlUwdGx5MFEyclhOS3dNVmIwR3piTjhvaE` +
		`cvTUV0R1JnTWF0SjN6b3RIM1dqeC9PeExKeTd5WTJUVHFxL3oKRWlKd0czTDM0cWF5Nm9SaWdPUis0U1UzdzZVYzJKUUs1MzR0aXdJREFRQUJB` +
		`b0lCQVFET2xyRE9RQ0NnT2JsMQo5VWMrLy84QkFrWksxZExyODc5UFNacGhCNkRycTFqeld6a3RzNEprUHZKTGR2VTVDMlJMTGQ0WjdmS0t4UH` +
		`U4CjZuZy8xSzhsMC85UTZHL3puME1kK1B4R2dBSjYvbHFPNFJTTlZGVGdWVFRXRm9pZEQvZ1ljYjFrRDRsaCtuZTIKRG1uemtWQU40MU90Tlp4` +
		`K0g3RVJEZUpwRTdoenFSOEhodnhxZU82Z25CMXJkZ3JRSE9MV1lSdmM1cGd2QS9BTwpYcTBRVXIrQWlUcTR0UW5oYjhDbDhJK2lLRmF5ZzZvY0` +
		`FnQXVCZkZBMnVBd29CL25LajZXTHlJVHV0NWE1VDBQCmxpbVJaYllGUTFyeHBJaVpUMmFja0NxUjN1Yk9qdVBGOCtJZHVWSmNXN05WcTFRSlls` +
		`RkFrSnVhTnpaRDlNMGkKUCs3WTgvTGhBb0dCQVBEYTg2cU9pazZpamNaajJtKzFub3dycnJINjdCRzhqRzdIYzJCZzU1M2VXWHZnQ3Z6RQppMk` +
		`xYU3J6VVV6SGN2aHFQRVZqV2RPbk1rVHkxK2VoZDRnV3FTZW9iUlFqcHAxYU40clA5dVcvOStZaHVoTlZWCnJ2QUh3ZHBTaTRlelovNEVERmxl` +
		`YUd5dXNWSkcvU1lJM096bnVQU051NW1lcysxN05Hb2pBZWtaQW9HQkFPUFYKMG5oRy9rNitQLzdlRXlqL2tjU3lPeUE5MzYvV05yVUU3bDF4b2` +
		`YyK3laSVVhUitOcE1manpmcVJqaitRWmZIZwpJS0kvYmJGWGtlWm9nWG5seHk0T1YvSmtKZy9oTHo2alJUQjhYTW9kbEhwVnFOaEZYcWJhV1Bj` +
		`a0h3WkhaVFU0CkNsQWg0QWZrZ2hpVWVrS2lhcTFNMWNyOE5CTWlyeTR2WWhKVXVReERBb0dCQUpyTG5aOFlUVHVNcmFHN3V6L2cKY2kyVVJZcU` +
		`53ZnNFT3gxWGdvZUd3RlZ0K2dUclVTUnpEVUpSSysrQVpwZTlUMUN5Y211dUtTVzZHLzN3MXRUSQp3ZUx5TnQ4Rzk2OXF1K21jOXY3SEtzOFhZ` +
		`N0NUbHp1ay9mRzJpcGhPUk83S0Z5UGlaaTFweDZOU0F4VG1HdnkrCjVYNDh6MW9kWFZ5MTZ0M09PVG1kbGpUQkFvR0FTYk5SY2pjRTdOUCtQNl` +
		`AyN3J3OW16Tk1qUkYyMnBxZzk4MncKamVuRVRTRDZjNWJHcXI1WEg1SkJmMXkyZHpsdXdOK1BydXgxdjNoa2FmUkViZm8yaEY5L2M1bVI5bkVS` +
		`cDJHSgpjRFhLamxjalFLK1UvdUR4eldlMGY3M2ZpMWh0Rk5vYisrLzVXSlJDd1ZER2UrZXVPb0V3WjRsT0R5S1pLSWVMCllnS21HYUVDZ1lBMF` +
		`prd3k5ejFXczRBTmpHK1lsYVV4cEtMY0pGZHlDSEtkRnI2NVdZc21HcU5rSmZHU0dlQjYKUkhNWk5Nb0RUUmhtaFFoajhNN04rRk10WkFVT01k` +
		`ZFovMWN2UkV0Rlc3KzY2dytYWnZqOUNRL3VlY3RwL3FiKwo2ZG5PYnJkbUxpWitVL056R0xLbUZnSlRjOVg3ZndtMTFQU2xpWkswV3JkblhLbn` +
		`praDlPaFE9PQotLS0tLUVORCBSU0EgUFJJVkFURSBLRVktLS0tLQo=`

	writeKeyToDiskDirs := func(t *testing.T, crt, key, dirCrt, dirKey string) (crtPath, keyPath string) {
		crtPath = filepath.Join(dirCrt, "tls.crt")
		keyPath = filepath.Join(dirKey, "tls.key")

		require.NoError(t, ioutil.WriteFile(crtPath, []byte(crt), 0600))
		require.NoError(t, ioutil.WriteFile(keyPath, []byte(key), 0600))
		return
	}

	writeKeyToDisk := func(t *testing.T, crt, key, dirName string) (crtPath, keyPath string) {
		return writeKeyToDiskDirs(t, crt, key, dirName, dirName)
	}

	t.Run("case=ensure generate is called if empty load", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		ev := make(EventChannel, 1)

		called := false
		err := errors.New("test")
		gen := CertificateGenerator(func() ([]tls.Certificate, error) {
			called = true
			return nil, err
		})

		p := NewProvider(ctx, ev)
		p.SetCertificatesGenerator(gen)

		require.Equal(t, err, p.LoadCertificates("", "", "", ""))
		assert.Equal(t, true, called)
	})

	t.Run("case=load certificate from files", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		ev := make(EventChannel, 1)

		dir, err := os.MkdirTemp(t.TempDir(), "test")
		require.NoError(t, err)

		crtPath, keyPath := writeKeyToDisk(t, k1crt, k1priv, dir)

		p := NewProvider(ctx, ev)

		require.NoError(t, p.LoadCertificates("", "", crtPath, keyPath))

		c, err := p.GetCertificate(nil)
		require.NoError(t, err)
		assert.NotEqual(t, nil, c)

		writeKeyToDisk(t, k2crt, k2priv, dir)
		assert.Equal(t, &ChangeEvent{}, <-ev)

		c2, err := p.GetCertificate(nil)
		require.NoError(t, err)
		assert.NotEqual(t, nil, c2)

		assert.NotEqual(t, 0, bytes.Compare(c.Certificate[0], c2.Certificate[0]))
	})

	t.Run("case=load certificate from files in two folders", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		ev := make(EventChannel, 1)

		dir1, err := os.MkdirTemp(t.TempDir(), "test")
		require.NoError(t, err)

		dir2, err := os.MkdirTemp(t.TempDir(), "test")
		require.NoError(t, err)

		crtPath, keyPath := writeKeyToDiskDirs(t, k1crt, k1priv, dir1, dir2)

		p := NewProvider(ctx, ev)

		require.NoError(t, p.LoadCertificates("", "", crtPath, keyPath))

		c, err := p.GetCertificate(nil)
		require.NoError(t, err)
		assert.NotEqual(t, nil, c)

		writeKeyToDiskDirs(t, k2crt, k2priv, dir1, dir2)
		time.Sleep(2 * time.Second)
		assert.Equal(t, &ChangeEvent{}, <-ev)

		c2, err := p.GetCertificate(nil)
		require.NoError(t, err)
		assert.NotEqual(t, nil, c2)

		assert.NotEqual(t, 0, bytes.Compare(c.Certificate[0], c2.Certificate[0]))
	})

	t.Run("case=load certificate base64", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		ev := make(EventChannel, 1)

		p := NewProvider(ctx, ev)

		require.NoError(t, p.LoadCertificates(certFixture, keyFixture, "", ""))

		c, err := p.GetCertificate(nil)
		require.NoError(t, err)
		assert.NotEqual(t, nil, c)
		assert.Equal(t, 0, len(ev))
	})

	t.Run("case=load certificate from files then base64 then files and check watcher", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		ev := make(EventChannel, 1)

		dir, err := os.MkdirTemp(t.TempDir(), "test")
		require.NoError(t, err)

		crtPath, keyPath := writeKeyToDisk(t, k1crt, k1priv, dir)

		p := NewProvider(ctx, ev)

		require.NoError(t, p.LoadCertificates("", "", crtPath, keyPath))

		c, err := p.GetCertificate(nil)
		require.NoError(t, err)
		assert.NotEqual(t, nil, c)

		require.NoError(t, p.LoadCertificates(certFixture, keyFixture, "", ""))

		c2, err := p.GetCertificate(nil)
		require.NoError(t, err)
		assert.NotEqual(t, nil, c2)
		assert.NotEqual(t, 0, bytes.Compare(c.Certificate[0], c2.Certificate[0]))

		// Using another temp dir to ensure watcher is working on change
		dir, err = os.MkdirTemp(t.TempDir(), "test")
		require.NoError(t, err)

		crtPath, keyPath = writeKeyToDisk(t, k1crt, k1priv, dir)

		require.NoError(t, p.LoadCertificates("", "", crtPath, keyPath))

		c3, err := p.GetCertificate(nil)
		require.NoError(t, err)
		assert.NotEqual(t, nil, c3)
		assert.Equal(t, 0, bytes.Compare(c.Certificate[0], c3.Certificate[0]))

		writeKeyToDisk(t, k2crt, k2priv, dir)
		assert.Equal(t, &ChangeEvent{}, <-ev)
	})
}
