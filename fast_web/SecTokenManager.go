package fast_web

import (
	"github.com/allegro/bigcache"
	"github.com/tdwu/fast_go/fast_base"
	"github.com/tdwu/fast_go/fast_utils"
	"os"
	"strconv"
	"strings"
	"time"
)

type SecToken struct {
	AccessToken  string `json:"accessToken" form:"访问令牌"`
	RefreshToken string `json:"refreshToken" form:"刷新令牌"`
	AppKey       string `json:"appKey"`
	UserId       int64  `json:"userId"`
	Data         string `json:"data"`
	CreateTime   string `json:"createTime"`
	ExpireTime   string `json:"expireTime"`
}

func (t *SecToken) IsValid() bool {
	// 当前时间在前，所以可用
	b := time.Now().Before(fast_utils.ToTime(t.ExpireTime))
	return b
}

var SecTokenController = SecTokenManager{}

type SecTokenManager struct {
	cacheInstance *bigcache.BigCache
	duration      time.Duration // 持续时间，秒
}

func (t *SecTokenManager) saveCacheToFile() error {
	items := make(map[string][]byte)
	iterator := t.cacheInstance.Iterator()

	for iterator.SetNext() {
		entry, err := iterator.Value()
		if err != nil {
			return err
		}
		items[entry.Key()] = entry.Value()
	}

	data, err := fast_base.Json.Marshal(items)
	if err != nil {
		return err
	}

	execPath := strings.ReplaceAll(fast_base.ExecPath(), "\\", "/")
	fast_base.Logger.Info("token_cache持久化保存:" + strconv.Itoa(len(items)))
	return os.WriteFile(execPath+"/cache/sec_token.bin", data, 0644)
}

func (t *SecTokenManager) loadFromFile() {
	execPath := strings.ReplaceAll(fast_base.ExecPath(), "\\", "/")
	dir := execPath + "/cache"
	// 检查目录是否存在
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// 目录不存在，创建它
		err := os.Mkdir(dir, 0755)
		if err != nil {
			fast_base.Logger.Error("创建目录失败:" + err.Error())
			return
		}
		fast_base.Logger.Info("创建目录成功:" + dir)
	}

	data, err := os.ReadFile(execPath + "/cache/sec_token.bin")
	if err != nil {
		fast_base.Logger.Info(execPath + "/cache/sec_token.bin")
		fast_base.Logger.Info("token_cache读取失败" + err.Error())
		return
	}

	items := make(map[string][]byte)
	if err := fast_base.Json.Unmarshal(data, &items); err != nil {
		return
	}
	for key, value := range items {
		t.cacheInstance.Set(key, value)
	}

	fast_base.Logger.Info("token_cache从文件中恢复" + strconv.Itoa(len(items)))
}

func (t *SecTokenManager) Init() *SecTokenManager {

	d, e := strconv.Atoi(ConfigServer.Session.Duration)
	if e != nil {
		d = 60 // 默认60分钟,和配置文件的默认值对齐
		fast_base.Logger.Error("转换异常")
	}
	fast_base.Logger.Info("token manager 过期时间：" + strconv.Itoa(d) + " 分钟")
	t.duration = time.Duration(d) * time.Minute
	cache, _ := bigcache.NewBigCache(bigcache.DefaultConfig(t.duration))
	t.cacheInstance = cache

	// 先从文件恢复
	t.loadFromFile()
	go func() {
		for {
			// 定时刷盘
			time.Sleep(time.Second * 10)
			t.saveCacheToFile()
		}
	}()
	return t
}

func (t *SecTokenManager) CreateNewToken(appKey string, userId int64, data string) *SecToken {
	// 1 立马挤掉用户之前的登录的（如果之前登录过），0代表立马
	t.clearUserToken(appKey, userId, 0)

	// 2 创建新token
	token := t.createToken(appKey, userId, data)

	// 3 记录用户当前的token
	t.cacheInstance.Set(appKey+"_"+"access_user_"+strconv.FormatInt(userId, 10), []byte(token.AccessToken))
	t.cacheInstance.Set(appKey+"_"+"refresh_user_"+strconv.FormatInt(userId, 10), []byte(token.RefreshToken))
	return token
}

func (t *SecTokenManager) RefreshNewToken(oldToken SecToken, data string) *SecToken {
	// 1 标记处理旧token， 1代表10秒后再删除
	t.clearUserToken(oldToken.AppKey, oldToken.UserId, 1)

	// 2 创建新token
	token := t.createToken(oldToken.AppKey, oldToken.UserId, data)

	// 3 记录用户当前的token
	t.cacheInstance.Set(oldToken.AppKey+"_"+"access_user_"+strconv.FormatInt(oldToken.UserId, 10), []byte(token.AccessToken))
	t.cacheInstance.Set(oldToken.AppKey+"_"+"refresh_user_"+strconv.FormatInt(oldToken.UserId, 10), []byte(token.RefreshToken))
	return token
}

func (t *SecTokenManager) createToken(appKey string, userId int64, data string) *SecToken {

	n1 := time.Now()
	n2 := time.Now().Add(t.duration)
	token := SecToken{
		AccessToken:  fast_utils.GetUUIDStr(),
		RefreshToken: fast_utils.GetUUIDStr(),
		AppKey:       appKey,
		UserId:       userId,
		Data:         data,
		CreateTime:   fast_utils.GetTimeStr(&n1),
		ExpireTime:   fast_utils.GetTimeStr(&n2),
	}
	d, _ := fast_base.Json.Marshal(token)
	// 存储下来
	t.cacheInstance.Set(appKey+"_"+"access_"+token.AccessToken, d)
	t.cacheInstance.Set(appKey+"_"+"refresh_"+token.RefreshToken, d)
	return &token
}

func (t *SecTokenManager) clearUserToken(appKey string, userId int64, mode int) {
	// 获取用户当前的token
	d1, e1 := t.cacheInstance.Get(appKey + "_" + "access_user_" + fast_utils.IntToStr(userId))
	d2, e2 := t.cacheInstance.Get(appKey + "_" + "refresh_user_" + fast_utils.IntToStr(userId))

	if e1 == nil {
		if d1 != nil {
			if mode == 0 {
				// 立即删除access_token
				t.cacheInstance.Delete(appKey + "_" + "access_" + string(d1))
			} else if mode == 1 {
				token := t.GetAccessToken(appKey, string(d1))
				if token != nil {
					// 保留，设置过期时间为10秒后
					n2 := time.Now().Add(time.Second * 10)
					token.ExpireTime = fast_utils.GetTimeStr(&n2)
					// 修改后，重新写回去
					d, _ := fast_base.Json.Marshal(token)
					t.cacheInstance.Set(appKey+"_"+"access_"+token.AccessToken, d)
				}
			}
		}
	}
	if e2 == nil {
		if d2 != nil {
			// 立即删除refresh_token
			t.cacheInstance.Delete(appKey + "_" + "refresh_" + string(d2))
		}
	}
}

func (t *SecTokenManager) GetAccessToken(appKey string, token string) *SecToken {
	return t.getToken(appKey, "access_", token)
}

func (t *SecTokenManager) GetRefreshToken(appKey string, token string) *SecToken {
	return t.getToken(appKey, "refresh_", token)
}

func (t *SecTokenManager) getToken(appKey string, prefix string, token string) *SecToken {
	key := appKey + "_" + prefix + token
	d, e := t.cacheInstance.Get(key)
	if e == nil {
		if d != nil {
			token := SecToken{}
			fast_base.Json.Unmarshal(d, &token)
			// 自动完成过期校验，如果过期了，则不返回。
			if !token.IsValid() {
				t.cacheInstance.Delete(key)
				return nil
			}
			return &token
		}
	}
	return nil
}
