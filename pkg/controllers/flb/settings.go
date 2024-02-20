package flb

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	fctx "github.com/flomesh-io/fsm/pkg/context"

	"k8s.io/apimachinery/pkg/labels"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/go-resty/resty/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/utils"
)

// Setting is the setting for a FLB instance per namespace
type Setting struct {
	httpClient            *resty.Client
	flbUser               string
	flbPassword           string
	k8sCluster            string
	flbDefaultAddressPool string
	flbDefaultAlgo        string
	token                 string
	hash                  string
	mu                    sync.Mutex
}

func (s *Setting) UpdateToken() error {
	return s.login(false)
}

func (s *Setting) ForceUpdateToken() error {
	return s.login(true)
}

func (s *Setting) login(force bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if force || s.token == "" {
		resp, err := s.httpClient.R().
			SetHeader("Content-Type", "application/json").
			SetBody(AuthRequest{Identifier: s.flbUser, Password: s.flbPassword}).
			SetResult(&AuthResponse{}).
			Post(flbAuthAPIPath)

		if err != nil {
			log.Error().Msgf("error happened while trying to login FLB, %s", err.Error())
			return err
		}

		if resp.StatusCode() != http.StatusOK {
			log.Error().Msgf("FLB server responsed with StatusCode: %d", resp.StatusCode())
			return fmt.Errorf("StatusCode: %d", resp.StatusCode())
		}

		s.token = resp.Result().(*AuthResponse).Token
	}

	return nil
}

// SettingManager is the manager for FLB settings
type SettingManager struct {
	ctx      *fctx.ControllerContext
	settings map[string]*Setting
}

// NewSettingManager returns a new setting manager
func NewSettingManager(ctx *fctx.ControllerContext) *SettingManager {
	mc := ctx.Config
	if !mc.IsFLBEnabled() {
		panic("FLB is not enabled")
	}

	if mc.GetFLBSecretName() == "" {
		panic("FLB Secret Name is empty, it's required.")
	}

	settings := make(map[string]*Setting)

	// get default settings
	defaultSetting, err := defaultGlobalSetting(ctx.KubeClient, mc)
	if err != nil {
		panic(err)
	}
	settings[flbDefaultSettingKey] = defaultSetting

	secrets, err := ctx.KubeClient.CoreV1().
		Secrets(corev1.NamespaceAll).
		List(context.TODO(), metav1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", mc.GetFLBSecretName()),
			LabelSelector: labels.SelectorFromSet(
				map[string]string{constants.FLBConfigSecretLabel: "true"},
			).String(),
		})

	if err != nil {
		panic(err)
	}

	for _, secret := range secrets.Items {
		secret := secret // fix lint GO-LOOP-REF
		if mc.IsFLBStrictModeEnabled() {
			settings[secret.Namespace] = newSetting(&secret)
		} else {
			settings[secret.Namespace] = newOverrideSetting(&secret, defaultSetting)
		}
	}

	return &SettingManager{
		ctx:      ctx,
		settings: settings,
	}
}

// GetSetting returns the setting for the namespace
func (sm *SettingManager) GetSetting(namespace string) *Setting {
	return sm.settings[namespace]
}

// SetSetting sets the setting for the namespace
func (sm *SettingManager) SetSetting(namespace string, setting *Setting) {
	sm.settings[namespace] = setting
}

// GetDefaultSetting returns the default setting
func (sm *SettingManager) GetDefaultSetting() *Setting {
	return sm.settings[flbDefaultSettingKey]
}

//// UpdateToken updates the token for the namespace
//func (sm *SettingManager) UpdateToken(namespace string, token string) {
//	setting := sm.GetSetting(namespace)
//	if setting == nil {
//		return
//	}
//
//	setting.token = token
//	sm.SetSetting(namespace, setting)
//}

func defaultGlobalSetting(api kubernetes.Interface, mc configurator.Configurator) (*Setting, error) {
	secret, err := api.CoreV1().
		Secrets(mc.GetFSMNamespace()).
		Get(context.TODO(), mc.GetFLBSecretName(), metav1.GetOptions{})

	if err != nil {
		return nil, err
	}

	if !secretHasRequiredLabel(secret) {
		return nil, fmt.Errorf("secret %s/%s doesn't have required label %s=true", mc.GetFSMNamespace(), mc.GetFLBSecretName(), constants.FLBConfigSecretLabel)
	}

	log.Debug().Msgf("Found Secret %s/%s", mc.GetFSMNamespace(), mc.GetFLBSecretName())

	log.Debug().Msgf("FLB base URL = %q", string(secret.Data[constants.FLBSecretKeyBaseURL]))
	log.Debug().Msgf("FLB default Address Pool = %q", string(secret.Data[constants.FLBSecretKeyDefaultAddressPool]))

	return newSetting(secret), nil
}

func newSetting(secret *corev1.Secret) *Setting {
	return &Setting{
		httpClient:            newHTTPClient(string(secret.Data[constants.FLBSecretKeyBaseURL])),
		flbUser:               string(secret.Data[constants.FLBSecretKeyUsername]),
		flbPassword:           string(secret.Data[constants.FLBSecretKeyPassword]),
		k8sCluster:            string(secret.Data[constants.FLBSecretKeyK8sCluster]),
		flbDefaultAddressPool: string(secret.Data[constants.FLBSecretKeyDefaultAddressPool]),
		flbDefaultAlgo:        string(secret.Data[constants.FLBSecretKeyDefaultAlgo]),
		hash:                  fmt.Sprintf("%d", utils.GetSecretDataHash(secret)),
		token:                 "",
	}
}

func newOverrideSetting(secret *corev1.Secret, defaultSetting *Setting) *Setting {
	s := &Setting{
		hash:  fmt.Sprintf("%d-%s", utils.GetSecretDataHash(secret), defaultSetting.hash),
		token: "",
	}

	if len(secret.Data[constants.FLBSecretKeyBaseURL]) == 0 {
		s.httpClient = defaultSetting.httpClient
	} else {
		s.httpClient = newHTTPClient(string(secret.Data[constants.FLBSecretKeyBaseURL]))
	}

	if len(secret.Data[constants.FLBSecretKeyUsername]) == 0 {
		s.flbUser = defaultSetting.flbUser
	} else {
		s.flbUser = string(secret.Data[constants.FLBSecretKeyUsername])
	}

	if len(secret.Data[constants.FLBSecretKeyPassword]) == 0 {
		s.flbPassword = defaultSetting.flbPassword
	} else {
		s.flbPassword = string(secret.Data[constants.FLBSecretKeyPassword])
	}

	if len(secret.Data[constants.FLBSecretKeyK8sCluster]) == 0 {
		s.k8sCluster = defaultSetting.k8sCluster
	} else {
		s.k8sCluster = string(secret.Data[constants.FLBSecretKeyK8sCluster])
	}

	if len(secret.Data[constants.FLBSecretKeyDefaultAddressPool]) == 0 {
		s.flbDefaultAddressPool = defaultSetting.flbDefaultAddressPool
	} else {
		s.flbDefaultAddressPool = string(secret.Data[constants.FLBSecretKeyDefaultAddressPool])
	}

	if len(secret.Data[constants.FLBSecretKeyDefaultAlgo]) == 0 {
		s.flbDefaultAlgo = defaultSetting.flbDefaultAlgo
	} else {
		s.flbDefaultAlgo = string(secret.Data[constants.FLBSecretKeyDefaultAlgo])
	}

	return s
}

func newHTTPClient(baseURL string) *resty.Client {
	return resty.New().
		SetTransport(&http.Transport{
			DisableKeepAlives:  false,
			MaxIdleConns:       10,
			IdleConnTimeout:    60 * time.Second,
			DisableCompression: false,
		}).
		SetScheme("http").
		SetBaseURL(baseURL).
		SetTimeout(5 * time.Second).
		SetDebug(true).
		EnableTrace()
}

func secretHasRequiredLabel(secret *corev1.Secret) bool {
	if len(secret.Labels) == 0 {
		return false
	}

	value, ok := secret.Labels[constants.FLBConfigSecretLabel]
	if !ok {
		return false
	}

	return value == "true"
}

func isSettingChanged(secret *corev1.Secret, setting, defaultSetting *Setting, mc configurator.Configurator) bool {
	if mc.IsFLBStrictModeEnabled() {
		hash := fmt.Sprintf("%d", utils.GetSecretDataHash(secret))
		if hash != setting.hash {
			return true
		}
	} else {
		hash := fmt.Sprintf("%d-%s", utils.GetSecretDataHash(secret), defaultSetting.hash)
		if hash != setting.hash {
			return true
		}
	}

	return false
}
