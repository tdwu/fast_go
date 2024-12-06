package fast_web

import (
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	zhTranslations "github.com/go-playground/validator/v10/translations/zh"
	"reflect"
	"regexp"
	"strings"
)

type ValidatorMessages map[string]string
type Validator interface {
	GetMessages() ValidatorMessages
}

// 定义一个全局翻译器T
var trans ut.Translator
var Validate = validator.New()

func InitTrans() (err error) {
	zhT := zh.New()
	uni := ut.New(zhT, zhT)
	trans, _ = uni.GetTranslator("zh")
	err = zhTranslations.RegisterDefaultTranslations(Validate, trans)

	return
}

func GetErrorStr(r interface{}, errs error) (string, bool) {
	var eList []string
	if e, isValidatorErrors := errs.(validator.ValidationErrors); e != nil && isValidatorErrors {
		vd, isValidator := r.(Validator)
		for _, err := range errs.(validator.ValidationErrors) {
			if isValidator {
				if message, exist := vd.GetMessages()[err.Field()+"."+err.Tag()]; exist {
					eList = append(eList, message)
				} else {
					eList = append(eList, err.Translate(trans))
				}
			} else {
				eList = append(eList, err.Translate(trans))
			}
		}
		return strings.Join(eList, "; \r\n"), true
	}
	return "", false
}

func removeTopStruct(fields map[string]string) map[string]string {
	res := map[string]string{}
	for field, err := range fields {
		res[field[strings.Index(field, ".")+1:]] = err
	}
	return res
}

// 自定义校验函数
func passwordValidation(fl validator.FieldLevel) bool {
	password := fl.Field().String()
	// 密码必须包含大小写字母和数字，且长度至少为 8
	match, _ := regexp.MatchString(`^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)[a-zA-Z\d]{8,}$`, password)
	return match
}

func LoadValidator() {
	InitTrans()
	Validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("tag"), ",", 2)[0]
		//if name == "" {
		//	name = strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		//	if name == "-" {
		//		//	return fld.Name
		//	}
		//}

		return name
	})
	Validate.RegisterValidation("password", passwordValidation)
	Validate.RegisterTranslation("password", trans, func(ut ut.Translator) error {
		return ut.Add("password", "{0}复杂度太低!", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("password", fe.Field())
		return t
	})
}
