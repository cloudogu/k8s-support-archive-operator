package collector

import (
	"context"
	_ "embed"
	"testing"
	"time"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

//go:embed testdata/secrets/secret.yaml
var secretYamlValuesBytes []byte

//go:embed testdata/secrets/yaml-secret.yaml
var yamlSecretYamlValuesBytes []byte

//go:embed testdata/secrets/json-secret.yaml
var jsonSecretYamlValuesBytes []byte

//go:embed testdata/secrets/nested-secret-with-json-and-yaml-arrays.yaml
var nestedSecretYamlValuesBytes []byte

func TestSecretCollector_NewSecretCollector(t *testing.T) {
	// given
	coreV1InterfaceMock := newMockCoreV1Interface(t)

	//when
	sc := NewSecretCollector(coreV1InterfaceMock)

	// then
	assert.NotNil(t, sc)
}

func TestSecretCollector_Name(t *testing.T) {
	// given
	coreV1InterfaceMock := newMockCoreV1Interface(t)
	sc := NewSecretCollector(coreV1InterfaceMock)

	//when
	name := sc.Name()

	// then
	assert.Equal(t, "Resources/Secrets", name)
}

func TestSecretsCollector_Collect(t *testing.T) {
	now := time.Now()

	type fields struct {
		coreV1Interface func(t *testing.T) coreV1Interface
	}
	type args struct {
		ctx          context.Context
		namespace    string
		start        time.Time
		end          time.Time
		resultChan   chan *domain.SecretYaml
		waitForClose bool
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantErr  func(t *testing.T, err error)
		wantData *domain.SecretYaml
	}{
		{
			name: "should fail to list secrets",
			fields: fields{
				coreV1Interface: func(t *testing.T) coreV1Interface {
					return createSecretInterfaceMock(t, [][]byte{}, assert.AnError)
				},
			},
			args: args{
				ctx:        testCtx,
				namespace:  testNamespace,
				start:      time.Time{},
				end:        time.Time{},
				resultChan: make(chan *domain.SecretYaml),
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "error listing secrets")
			},
		},
		{
			name: "should just close the channel if no secrets are fetched",
			fields: fields{
				coreV1Interface: func(t *testing.T) coreV1Interface {
					return createSecretInterfaceMock(t, [][]byte{}, nil)
				},
			},
			args: args{
				ctx:        testCtx,
				namespace:  testNamespace,
				start:      time.Time{},
				end:        time.Time{},
				resultChan: make(chan *domain.SecretYaml),
			},
			wantErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "should write flat secret to channel and close",
			fields: fields{
				coreV1Interface: func(t *testing.T) coreV1Interface {
					return createSecretInterfaceMock(t, [][]byte{secretYamlValuesBytes}, nil)
				},
			},
			args: args{
				ctx:          testCtx,
				namespace:    testNamespace,
				start:        time.Time{},
				end:          now,
				resultChan:   make(chan *domain.SecretYaml),
				waitForClose: true,
			},
			wantErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
			wantData: &domain.SecretYaml{
				ApiVersion: "v1",
				Kind:       "Secret",
				SecretType: "Opaque",
				Data:       map[string]string{"password": "***", "username": "***"},
				Metadata: domain.SecretYamlMetaData{
					Name:              "sensitive-config",
					Namespace:         "default",
					CreationTimestamp: "2025-08-11 16:16:25 +0200 CEST",
					UID:               "c8d6e45f-3e41-4829-86ac-1227a7c2f112",
					Labels:            map[string]string{"app": "ces", "dogu.name": "test-dogu"},
				},
			},
		},
		{
			name: "should write yaml Secret to channel and close",
			fields: fields{
				coreV1Interface: func(t *testing.T) coreV1Interface {
					return createSecretInterfaceMock(t, [][]byte{yamlSecretYamlValuesBytes}, nil)
				},
			},
			args: args{
				ctx:          testCtx,
				namespace:    testNamespace,
				start:        time.Time{},
				end:          now,
				resultChan:   make(chan *domain.SecretYaml),
				waitForClose: true,
			},
			wantErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
			wantData: &domain.SecretYaml{
				ApiVersion: "v1",
				Kind:       "Secret",
				SecretType: "Opaque",
				Data:       map[string]string{"config.yaml": "sa-dogu:\n    password: '***'\n    username: '***'\n"},
				Metadata: domain.SecretYamlMetaData{
					Name:              "sensitive-config",
					Namespace:         "default",
					CreationTimestamp: "2025-08-11 16:16:25 +0200 CEST",
					UID:               "f4c9b4c2-73a8-48a5-bd92-fd4e5c236c87",
					Labels:            map[string]string{"app": "ces", "dogu.name": "test-dogu"},
				},
			},
		},
		{
			name: "should write json secret to channel and close",
			fields: fields{
				coreV1Interface: func(t *testing.T) coreV1Interface {
					return createSecretInterfaceMock(t, [][]byte{jsonSecretYamlValuesBytes}, nil)
				},
			},
			args: args{
				ctx:          testCtx,
				namespace:    testNamespace,
				start:        time.Time{},
				end:          now,
				resultChan:   make(chan *domain.SecretYaml),
				waitForClose: true,
			},
			wantErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
			wantData: &domain.SecretYaml{
				ApiVersion: "v1",
				Kind:       "Secret",
				SecretType: "Opaque",
				Data:       map[string]string{"config.json": "{\"auths\":{\"localhost\":{\"password\":\"***\",\"username\":\"***\"}}}"},
				Metadata: domain.SecretYamlMetaData{
					Name:              "sensitive-config",
					Namespace:         "default",
					CreationTimestamp: "2025-08-11 16:16:25 +0200 CEST",
					UID:               "8b1a8a6e-bd7a-4e7f-9a51-f5b9a6cfc20b",
					Labels:            map[string]string{"app": "ces", "dogu.name": "test-dogu"},
				},
			},
		},
		{
			name: "should write nested secret to channel and close",
			fields: fields{
				coreV1Interface: func(t *testing.T) coreV1Interface {
					return createSecretInterfaceMock(t, [][]byte{nestedSecretYamlValuesBytes}, nil)
				},
			},
			args: args{
				ctx:          testCtx,
				namespace:    testNamespace,
				start:        time.Time{},
				end:          now,
				resultChan:   make(chan *domain.SecretYaml),
				waitForClose: true,
			},
			wantErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
			wantData: &domain.SecretYaml{
				ApiVersion: "v1",
				Kind:       "Secret",
				SecretType: "Opaque",
				Data:       map[string]string{"config.json": "{\"groups\":[{\"name\":\"***\",\"rule\":\"***\"},{\"name\":\"***\",\"rule\":\"***\"},{\"name\":\"***\",\"rule\":\"***\"}]}", "config.yaml": "groups:\n    - name: '***'\n      rule: '***'\n    - name: '***'\n      rule: '***'\n    - name: '***'\n      rule: '***'\n", "password": "***", "username": "***"},
				Metadata: domain.SecretYamlMetaData{
					Name:              "sensitive-config",
					Namespace:         "default",
					UID:               "0f1e4b3c-9d89-4b28-94b4-1df1e5e0cb5c",
					CreationTimestamp: "2025-08-11 16:16:25 +0200 CEST",
					Labels:            map[string]string{"app": "ces", "dogu.name": "test-dogu"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &SecretCollector{
				coreV1Interface: tt.fields.coreV1Interface(t),
			}

			group, _ := errgroup.WithContext(tt.args.ctx)
			group.Go(func() error {
				err := sc.Collect(tt.args.ctx, tt.args.namespace, tt.args.start, tt.args.end, tt.args.resultChan)
				return err
			})

			if tt.args.waitForClose {
				timer := time.NewTimer(time.Second * 2)
				group.Go(func() error {
					<-timer.C
					defer func() {
						// recover panic if the channel is closed correctly from the test
						if r := recover(); r != nil {
							tt.args.resultChan <- nil
							return
						}
					}()

					return nil
				})

				v, open := <-tt.args.resultChan
				if v == nil && open {
					t.Fatal("test timed out")
				}

				if v != nil && open {
					assert.Equal(t, tt.wantData, v)
				}
			}

			err := group.Wait()
			tt.wantErr(t, err)
		})
	}
}

func createSecretInterfaceMock(t *testing.T, secretsBytes [][]byte, expectedError error) coreV1Interface {
	secrets := make([]corev1.Secret, 0)
	for _, secretsByte := range secretsBytes {
		var secret corev1.Secret
		err := yaml.Unmarshal(secretsByte, &secret)
		require.NoError(t, err)
		secrets = append(secrets, secret)
	}
	secretList := &corev1.SecretList{
		Items: secrets,
	}

	labelSelector := "app=ces"

	clientMock := newMockSecretInterface(t)
	clientMock.EXPECT().List(testCtx, metav1.ListOptions{LabelSelector: labelSelector}).Return(secretList, expectedError)

	interfaceMock := newMockCoreV1Interface(t)
	interfaceMock.EXPECT().Secrets(testNamespace).Return(clientMock)

	return interfaceMock
}
