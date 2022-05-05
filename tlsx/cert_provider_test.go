package tlsx

// func TestCertProvider(t *testing.T) {
// 	tmpDir, err := ioutil.TempDir(os.TempDir(), "test_cert_provider")
// 	require.NoError(t, err)
//
// 	defer os.RemoveAll(tmpDir)
//
// 	logger := logrusx.New("", "")
// 	cert1Data := []byte("cert1")
// 	cert1 := tls.Certificate{Certificate: [][]byte{cert1Data}}
// 	cert2Data := []byte("cert2")
// 	cert2 := tls.Certificate{Certificate: [][]byte{cert2Data}}
//
// 	certLoad := CertLoadFunc(func() ([]tls.Certificate, error) {
// 		return []tls.Certificate{cert2}, nil
// 	})
//
// 	crtLoc := CertLocation{
// 		KeyPath:  path.Join(tmpDir, "tls.key"),
// 		CertPath: path.Join(tmpDir, "tls.crt"),
// 	}
// 	provider := NewCertProvider(
// 		[]tls.Certificate{cert1},
// 		crtLoc,
// 		certLoad,
// 		logger,
// 	)
// 	defer provider.Stop()
//
// 	//Checking initial cert load
// 	cert, err := provider.GetCertificate(nil)
// 	require.NoError(t, err)
// 	assert.Equal(t, cert1Data, cert.Certificate[0])
//
// 	//Touching tls.key to trigger fs change
// 	_, err = os.Create(crtLoc.KeyPath)
// 	require.NoError(t, err)
//
// 	//The first tls.Certificate should stay before the reloading process
// 	cert, err = provider.GetCertificate(nil)
// 	require.NoError(t, err)
// 	assert.Equal(t, cert1Data, cert.Certificate[0])
//
// 	//New cert should be loaded
// 	time.Sleep(3 * time.Second)
// 	cert, err = provider.GetCertificate(nil)
// 	require.NoError(t, err)
// 	assert.Equal(t, cert2Data, cert.Certificate[0])
// }
//
// func TestCertProviderDualDir(t *testing.T) {
// 	tmpDir, err := ioutil.TempDir(os.TempDir(), "test_cert_provider")
// 	require.NoError(t, err)
// 	defer os.RemoveAll(tmpDir)
//
// 	tmpDir2, err := ioutil.TempDir(os.TempDir(), "test_cert_provider")
// 	require.NoError(t, err)
// 	defer os.RemoveAll(tmpDir2)
//
// 	logger := logrusx.New("", "")
// 	cert1Data := []byte("cert1")
// 	cert1 := tls.Certificate{Certificate: [][]byte{cert1Data}}
// 	cert2Data := []byte("cert2")
// 	cert2 := tls.Certificate{Certificate: [][]byte{cert2Data}}
//
// 	certLoad := CertLoadFunc(func() ([]tls.Certificate, error) {
// 		return []tls.Certificate{cert2}, nil
// 	})
//
// 	crtLoc := CertLocation{
// 		KeyPath:  path.Join(tmpDir, "tls.key"),
// 		CertPath: path.Join(tmpDir2, "tls.crt"),
// 	}
// 	provider := NewCertProvider(
// 		[]tls.Certificate{cert1},
// 		crtLoc,
// 		certLoad,
// 		logger,
// 	)
// 	defer provider.Stop()
//
// 	//Checking initial cert load
// 	cert, err := provider.GetCertificate(nil)
// 	require.NoError(t, err)
// 	assert.Equal(t, cert1Data, cert.Certificate[0])
//
// 	//Touching tls.crt to trigger fs change
// 	_, err = os.Create(crtLoc.CertPath)
// 	require.NoError(t, err)
//
// 	//The first tls.Certificate should stay before the reloading process
// 	cert, err = provider.GetCertificate(nil)
// 	require.NoError(t, err)
// 	assert.Equal(t, cert1Data, cert.Certificate[0])
//
// 	//New cert should be loaded
// 	time.Sleep(3 * time.Second)
// 	cert, err = provider.GetCertificate(nil)
// 	require.NoError(t, err)
// 	assert.Equal(t, cert2Data, cert.Certificate[0])
// }
//
// func TestCertProviderErrorReload(t *testing.T) {
// 	tmpDir, err := ioutil.TempDir(os.TempDir(), "test_cert_provider")
// 	require.NoError(t, err)
//
// 	defer os.RemoveAll(tmpDir)
//
// 	logger := logrusx.New("", "")
// 	cert1Data := []byte("cert1")
// 	cert1 := tls.Certificate{Certificate: [][]byte{cert1Data}}
//
// 	certLoad := CertLoadFunc(func() ([]tls.Certificate, error) {
// 		return nil, errors.New("Test error")
// 	})
//
// 	crtLoc := CertLocation{
// 		KeyPath:  path.Join(tmpDir, "tls.key"),
// 		CertPath: path.Join(tmpDir, "tls.crt"),
// 	}
// 	provider := NewCertProvider(
// 		[]tls.Certificate{cert1},
// 		crtLoc,
// 		certLoad,
// 		logger,
// 	)
// 	defer provider.Stop()
//
// 	//Checking initial cert load
// 	cert, err := provider.GetCertificate(nil)
// 	require.NoError(t, err)
// 	assert.Equal(t, cert1Data, cert.Certificate[0])
//
// 	//Touching tls.key to trigger fs change
// 	_, err = os.Create(crtLoc.KeyPath)
// 	require.NoError(t, err)
//
// 	//Old cert should stay
// 	time.Sleep(3 * time.Second)
// 	cert, err = provider.GetCertificate(nil)
// 	require.NoError(t, err)
// 	assert.Equal(t, cert1Data, cert.Certificate[0])
// }
