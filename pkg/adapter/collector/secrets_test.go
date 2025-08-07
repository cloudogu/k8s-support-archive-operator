package collector

import (
	_ "embed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
)

func createTestSecrets() *corev1.SecretList {
	secrets := make([]corev1.Secret, 0)
	jsonContent := "|2-\n"
	yamlContent := "|\n"
	jsonDataMap := make(map[string][]byte)
	yamlDataMap := make(map[string][]byte)
	flatDataMap := make(map[string][]byte)
	// TODO decode
	jsonDataMap["config.json"] = []byte(jsonContent)
	yamlDataMap["config.yaml"] = []byte(yamlContent)
	flatDataMap["username"] = []byte("admin")
	flatDataMap["password"] = []byte("admin")

	yamlDataSecret := &corev1.Secret{
		Type: corev1.SecretTypeOpaque,
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      "yaml-content-secret",
		},
		Data: yamlDataMap,
	}
	jsonDataSecret := &corev1.Secret{
		Type: corev1.SecretTypeOpaque,
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      "json-content-secret",
		},
		Data: jsonDataMap,
	}
	flatDataSecret := &corev1.Secret{
		Type: corev1.SecretTypeOpaque,
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      "flat-content-secret",
		},
		Data: flatDataMap,
	}
	secrets = append(secrets, *yamlDataSecret)
	secrets = append(secrets, *jsonDataSecret)
	secrets = append(secrets, *flatDataSecret)
	return &corev1.SecretList{Items: secrets}
}

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
	assert.Equal(t, "Secret", name)
}

func TestSecretCollector_Collect(t *testing.T) {
	tests := []struct {
		name              string
		coreV1InterfaceFn func(t *testing.T) coreV1Interface
		resultChanFn      func(t *testing.T) chan *corev1.SecretList
		wantData          *corev1.Secret
		waitForClose      bool
		wantErr           func(t *testing.T, err error)
	}{
		{
			name: "should fail to list secrets",
			coreV1InterfaceFn: func(t *testing.T) coreV1Interface {
				clientMock := newMockSecretInterface(t)
				clientMock.EXPECT().List(testCtx, metav1.ListOptions{LabelSelector: "app=ces"}).Return(nil, assert.AnError)

				interfaceMock := newMockCoreV1Interface(t)
				interfaceMock.EXPECT().Secrets(testNamespace).Return(clientMock)

				return interfaceMock
			},
			resultChanFn: func(t *testing.T) chan *corev1.SecretList {
				return make(chan *corev1.SecretList)
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				require.ErrorIs(t, err, assert.AnError)
				require.ErrorContains(t, err, "error listing secrets")
			},
		},
		{
			name: "should return an empty channel of secrets",
			coreV1InterfaceFn: func(t *testing.T) coreV1Interface {
				clientMock := newMockSecretInterface(t)
				clientMock.EXPECT().List(testCtx, metav1.ListOptions{LabelSelector: "app=ces"}).Return(&corev1.SecretList{Items: make([]corev1.Secret, 0)}, nil)

				interfaceMock := newMockCoreV1Interface(t)
				interfaceMock.EXPECT().Secrets(testNamespace).Return(clientMock)

				return interfaceMock
			},
			resultChanFn: func(t *testing.T) chan *corev1.SecretList {
				return make(chan *corev1.SecretList)
			},
			wantErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "successfully censor all types of secrets",
			coreV1InterfaceFn: func(t *testing.T) coreV1Interface {
				secrets := createTestSecrets()
				clientMock := newMockSecretInterface(t)
				clientMock.EXPECT().List(testCtx, metav1.ListOptions{LabelSelector: "app=ces"}).Return(secrets, nil)

				interfaceMock := newMockCoreV1Interface(t)
				interfaceMock.EXPECT().Secrets(testNamespace).Return(clientMock)

				return interfaceMock
			},
			resultChanFn: func(t *testing.T) chan *corev1.SecretList {
				return make(chan *corev1.SecretList)
			},
			waitForClose: true,
			wantErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
			wantData: &corev1.Secret{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &SecretCollector{
				coreV1Interface: tt.coreV1InterfaceFn(t),
			}
			timestamp := time.Time{}
			group, _ := errgroup.WithContext(testCtx)
			group.Go(func() error {
				err := sc.Collect(testCtx, testNamespace, timestamp, timestamp, tt.resultChanFn(t))
				return err
			})

			if tt.waitForClose {
				timer := time.NewTimer(time.Second * 2)
				group.Go(func() error {
					<-timer.C
					defer func() {
						// recover panic if the channel is closed correctly from the test
						if r := recover(); r != nil {
							tt.resultChanFn(t) <- nil
							return
						}
					}()

					return nil
				})

				v, open := <-tt.resultChanFn(t)
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
