package fast_wgen

import (
	"container/list"
	"flag"
	"fmt"
	"github.com/swaggo/swag"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var dir = flag.String("d", "./", "扫描路径")
var file = flag.String("s", "./LoadRouter.go", "生成的路由文件保存地址")
var wrapper = flag.String("w", "1", "是否包装一层(内置封装)")

// go get -u github.com/swaggo/swag/cmd/swag
// go install github.com/swaggo/swag/cmd/swag@latest
// go get -u github.com/thinkhp/swag/cmd/swag
// go get -u github.com/tdwu/swag/cmd/swag
// go install gr.go
// -g D:\ws_go\go_tpl\src\sys\SysApplication.go --ot json  -o D:\ws_go\go_tpl\static\doc\json
func main() {
	flag.Parse()
	genDir, genOutput, genWrapper := getParamFromMain()
	if len(genDir) == 0 {
		genDir = *dir
	}
	if len(genOutput) == 0 {
		genOutput = *file
	}
	if len(genWrapper) == 0 {
		genWrapper = *wrapper
	}
	fmt.Println("扫描路径：" + genDir)
	fmt.Println("输出地址：" + genOutput)
	fmt.Println("内置封装：" + genWrapper)
	MakeRouter(strings.Split(genDir, ","), genOutput, genWrapper)
}

func MakeRouter(searchDirs []string, outputFile string, w string) {
	for i := range searchDirs {
		searchDirs[i] = strings.Trim(searchDirs[i], " ")
		_, err := os.Stat(searchDirs[i])
		if os.IsNotExist(err) {
			fmt.Println("目录不存在：" + searchDirs[i])
			return
		}
	}

	c := Collector{packages: NewPackagesDefinitions(), Routers: list.New()}
	c.ParseAPIMultiSearchDir(searchDirs)

	fmt.Println("【阶段3】-----------------------")
	fmt.Println("【阶段3】生成文件：" + outputFile)
	os.Remove(outputFile)
	// 新建文件
	file, err := os.Create(outputFile)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()

	file.WriteString("package main\n\n")
	headerMap := map[string]string{}
	header := "\t\"github.com/gin-gonic/gin\"\n\t\"reflect\"\n\t\"fast_web\"\n"
	body := ""

	limits := map[string]Limit{}

	// 找到limit，并分组
	for i := c.Routers.Front(); i != nil; i = i.Next() {
		r, _ := i.Value.(*RouteProperties)
		if len(r.Limit.Name) > 0 {
			l, ok := limits[r.Limit.Name]
			if !ok {
				l = r.Limit
				limits[r.Limit.Name] = r.Limit
			} else {
				if len(l.Num) == 0 {
					l.Num = r.Limit.Num
					l.Cap = r.Limit.Cap
				}
			}
		}
	}

	// 生成limit代码
	for _, limit := range limits {
		body = body + fmt.Sprintf("\t%s := fast_web.RateLimitMiddleware(%s, %s)\n", limit.Name, limit.Num, limit.Cap)
	}

	// 生成路由，其中自动添加limit中间件
	for i := c.Routers.Front(); i != nil; i = i.Next() {
		r, _ := i.Value.(*RouteProperties)
		if len(headerMap[r.PackagePath]) == 0 {
			headerMap[r.PackagePath] = firstCharUpper(r.PackagePath)
			header = header + fmt.Sprintf("\t%v \"%v\"\n", headerMap[r.PackagePath], r.PackagePath)
		}

		var limit = ""
		if len(r.Limit.Name) > 0 {
			limit = r.Limit.Name + ", "
		} else if len(r.Limit.Num) > 0 && len(r.Limit.Cap) > 0 {
			limit = fmt.Sprintf("fast_web.RateLimitMiddleware(%s, %s), ", r.Limit.Num, r.Limit.Cap)
		}

		if len(r.Receiver) == 0 {
			if w == "1" {
				body = body + fmt.Sprintf("\tgin.%v(\"%v\", %vfast_web.GenHandlerFunc(reflect.ValueOf(%v.%v)))\n", strings.ToUpper(r.HTTPMethod), r.Path, limit, headerMap[r.PackagePath], r.MethodName)
			} else {
				body = body + fmt.Sprintf("\tgin.%v(\"%v\", %v%v.%v)\n", strings.ToUpper(r.HTTPMethod), r.Path, limit, headerMap[r.PackagePath], r.MethodName)
			}
		} else {
			if w == "1" {
				body = body + fmt.Sprintf("\tgin.%v(\"%v\", %vfast_web.GenHandlerFunc(reflect.ValueOf(%v.%v{}.%v)))\n", strings.ToUpper(r.HTTPMethod), r.Path, limit, headerMap[r.PackagePath], r.Receiver, r.MethodName)
			} else {
				body = body + fmt.Sprintf("\tgin.%v(\"%v\", %v%v.%v{}.%v)\n", strings.ToUpper(r.HTTPMethod), r.Path, limit, headerMap[r.PackagePath], r.Receiver, r.MethodName)
			}
		}

	}
	file.WriteString("import (\n" + header + ")\n\n")
	file.WriteString("func LoadRouters(gin *gin.Engine) {\n" + body + "}\n")
}

func IfStr(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}

func firstCharUpper(str string) string {
	// 先将.转成/,再统一分割
	ss := strings.Split(strings.ReplaceAll(str, ".", "/"), "/")
	for i, s2 := range ss {
		vv := []rune(s2)
		if len(vv) > 0 && vv[0] >= 97 && vv[0] <= 122 {
			vv[0] -= 32
			ss[i] = string(vv)
			continue
		}
		ss[i] = s2
	}
	return strings.Join(ss, "")
}

type Collector struct {
	excludes    map[string]struct{}
	ParseVendor bool
	packages    *PackagesDefinitions
	Routers     *list.List
}

// ParseAPIMultiSearchDir is like ParseAPI but for multiple search dirs.
// 【改造1】，分析入口
func (this *Collector) ParseAPIMultiSearchDir(searchDirs []string) error {
	fmt.Println("【阶段1】-----------------------")
	// 阶段1 扫描所有需要分析的文件
	for _, searchDir := range searchDirs {
		fmt.Printf("Generate general API Info, search dir:%s\n", searchDir)

		packageDir, err := getPkgName(searchDir)
		if err != nil {
			fmt.Printf("warning: failed to get package name in dir: %s, error: %s", searchDir, err.Error())
		}
		// AST处理
		err = this.getAllGoFileInfo(packageDir, searchDir)
		if err != nil {
			return err
		}
	}
	fmt.Println("【阶段2】-----------------------")
	// 阶段2 分析操作
	err := rangeFiles(this.packages.files, this.ParseRouterAPIInfo)
	if err != nil {
		return err
	}

	return nil
}

func getParamFromMain() (string, string, string) {
	dir, _ := os.ReadDir("./")
	for _, v := range dir {
		path := v.Name()
		if v.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(path), "_test.go") || filepath.Ext(path) != ".go" {
			continue
		}
		astFile, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
		if err != nil {
			continue
		}
		for _, astDescription := range astFile.Decls {
			astDeclaration, ok := astDescription.(*ast.FuncDecl)
			if astDeclaration == nil || astDeclaration.Name == nil {
				continue
			}
			if len(astDeclaration.Type.Params.List) == 0 && astDeclaration.Name.Name == "main" {
				if ok && astDeclaration.Doc != nil && astDeclaration.Doc.List != nil {
					genDir, genOutput, genWrapper := "", "", ""
					for _, comment := range astDeclaration.Doc.List {
						commentLine := strings.TrimSpace(strings.TrimLeft(comment.Text, "/"))
						attribute := strings.Fields(commentLine)[0]
						lineRemainder := strings.TrimSpace(commentLine[len(attribute):])
						lowerAttribute := strings.ToLower(attribute)
						if lowerAttribute == "@gendir" {
							genDir = lineRemainder
						}
						if lowerAttribute == "@genoutput" {
							genOutput = lineRemainder
						}
						if lowerAttribute == "@genwrapper" {
							genWrapper = lineRemainder
						}
					}
					if len(genDir) > 0 || len(genOutput) > 0 || len(genWrapper) > 0 {
						fmt.Println("找到主入口文件：" + path)
						return genDir, genOutput, genWrapper
					}
				}

			}
		}
	}
	return "", "", ""
}

// /////////////////////////////////////////阶段1 扫描所有需要分析的文件////////////////////////////////
// GetAllGoFileInfo gets all Go source files information for given searchDir.
func (this *Collector) getAllGoFileInfo(packageDir, searchDir string) error {
	return filepath.Walk(searchDir, func(path string, f os.FileInfo, _ error) error {
		if err := this.Skip(path, f); err != nil {
			return err
		} else if f.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(searchDir, path)
		if err != nil {
			return err
		}

		return this.parseFile(filepath.ToSlash(filepath.Dir(filepath.Clean(filepath.Join(packageDir, relPath)))), path, nil)
	})
}

func (this *Collector) parseFile(packageDir, path string, src interface{}) error {
	if strings.HasSuffix(strings.ToLower(path), "_test.go") || filepath.Ext(path) != ".go" {
		return nil
	}

	// positions are relative to FileSet
	astFile, err := parser.ParseFile(token.NewFileSet(), path, src, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("ParseFile error:%+v", err)
	}

	// t := astFile.Name.Name
	// fmt.Println("【阶段1】语法分析, 包：" + packageDir + "   路径：" + path + " " + t)

	fmt.Println("【阶段1】语法分析, 包：" + packageDir + "   路径：" + path)
	err = this.packages.CollectAstFile(packageDir, path, astFile)
	if err != nil {
		return err
	}

	return nil
}

func (this *Collector) Skip(path string, f os.FileInfo) error {
	return walkWith(this.excludes, this.ParseVendor)(path, f)
}

func walkWith(excludes map[string]struct{}, parseVendor bool) func(path string, fileInfo os.FileInfo) error {
	return func(path string, f os.FileInfo) error {
		if f.IsDir() {
			if !parseVendor && f.Name() == "vendor" || // ignore "vendor"
				f.Name() == "docs" || // exclude docs
				len(f.Name()) > 1 && f.Name()[0] == '.' { // exclude all hidden folder
				return filepath.SkipDir
			}

			if excludes != nil {
				if _, ok := excludes[path]; ok {
					return filepath.SkipDir
				}
			}
		}

		return nil
	}
}

func getPkgName(searchDir string) (string, error) {
	cmd := exec.Command("go", "list", "-f={{.ImportPath}}")
	cmd.Dir = searchDir
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("execute go list command, %s, stdout:%s, stderr:%s", err, stdout.String(), stderr.String())
	}

	outStr, _ := stdout.String(), stderr.String()

	if outStr[0] == '_' { // will shown like _/{GOPATH}/src/{YOUR_PACKAGE} when NOT enable GO MODULE.
		outStr = strings.TrimPrefix(outStr, "_"+build.Default.GOPATH+"/src/")
	}
	f := strings.Split(outStr, "\n")
	outStr = f[0]

	return outStr, nil
}

// /////////////////////////////////////////阶段2 分析文件中的路由信息：ParseRouterAPIInfo ////////////////////////////////
// RangeFiles for range the collection of ast.File in alphabetic order.
// 【改造2】，添加参数packagePath
func rangeFiles(files map[*ast.File]*swag.AstFileInfo, handle func(filename string, packagePath string, file *ast.File) error) error {
	sortedFiles := make([]*swag.AstFileInfo, 0, len(files))
	for _, info := range files {
		sortedFiles = append(sortedFiles, info)
	}

	sort.Slice(sortedFiles, func(i, j int) bool {
		return strings.Compare(sortedFiles[i].Path, sortedFiles[j].Path) < 0
	})

	for _, info := range sortedFiles {
		err := handle(info.Path, info.PackagePath, info.File)
		if err != nil {
			return err
		}
	}

	return nil
}

func findLimit(list []*ast.Comment) Limit {
	for _, comment := range list {
		commentLine := strings.TrimSpace(strings.TrimLeft(comment.Text, "/"))
		attribute := strings.Fields(commentLine)[0]
		lineRemainder := strings.TrimSpace(commentLine[len(attribute):])
		lowerAttribute := strings.ToLower(attribute)
		if lowerAttribute == "@limit" {
			ss := strings.Split(lineRemainder, " ")
			l := Limit{}
			for _, s := range ss {
				s = strings.Trim(s, " ")
				if len(s) > 0 {
					if l.Num == "" {
						l.Num = s
					} else if l.Cap == "" {
						l.Cap = s
					} else if l.Name == "" {
						l.Name = s
					}
				}
			}
			if len(l.Cap) == 0 && len(l.Num) > 0 {
				l.Cap = l.Num
			}
			return l
		}
	}
	return Limit{}
}

// ParseRouterAPIInfo parses router api info for given astFile.
// 【改造3】，提取每个接口上的注释，从而获取router信息，并定位出PackageName+MethodName+OperationName
func (this *Collector) ParseRouterAPIInfo(fileName string, packagePath string, astFile *ast.File) error {

	fmt.Println("【阶段2】--开始处理：" + fileName + "    " + packagePath)
	for _, astDescription := range astFile.Decls {
		astDeclaration, ok := astDescription.(*ast.FuncDecl)
		if astDeclaration == nil || astDeclaration.Name == nil {
			continue
		}
		//name := astFile.Name.Name + "." + astDeclaration.Name.Name
		if ok && astDeclaration.Doc != nil && astDeclaration.Doc.List != nil {
			// for per 'function' comment, create a new 'Operation' object
			for _, comment := range astDeclaration.Doc.List {
				if comment.Text == "//" {
					continue
				}
				commentLine := strings.TrimSpace(strings.TrimLeft(comment.Text, "/"))
				attribute := strings.Fields(commentLine)[0]
				lineRemainder := strings.TrimSpace(commentLine[len(attribute):])
				lowerAttribute := strings.ToLower(attribute)

				if lowerAttribute == "@router" {

					matches := routerPattern.FindStringSubmatch(lineRemainder)
					if len(matches) != 3 {
						panic(fmt.Sprintf("can not parse router comment \"%s\" in file \"%s\"", commentLine, fileName))
					}
					var router RouteProperties
					if astDeclaration.Recv != nil {
						n, e := astDeclaration.Recv.List[0].Type.(*ast.Ident)
						if e == false {
							n, e = astDeclaration.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident)
							if !e {
								fmt.Println(astDeclaration.Recv.List[0].Type)
							}
						}
						router = RouteProperties{
							Path:        matches[1],
							HTTPMethod:  strings.ToUpper(matches[2]),
							PackageName: astFile.Name.Name,
							PackagePath: packagePath,
							MethodName:  astDeclaration.Name.Name,
							Receiver:    n.Name,
							Limit:       findLimit(astDeclaration.Doc.List),
						}
					} else {
						router = RouteProperties{
							Path:        matches[1],
							HTTPMethod:  strings.ToUpper(matches[2]),
							PackageName: astFile.Name.Name,
							PackagePath: packagePath,
							MethodName:  astDeclaration.Name.Name,
							Limit:       findLimit(astDeclaration.Doc.List),
						}
					}

					fmt.Println("【阶段2】++找到接口：" + router.PackageName + "." + router.MethodName + " -> " + lineRemainder)
					this.Routers.PushBack(&router)

					break
				}
			}
		}
	}
	return nil
}

var routerPattern = regexp.MustCompile(`^(/[\w./\-{}+:$]*)[[:blank:]]+\[(\w+)]`)

// /////////////////////////////////////////开闭原则，无法直接受用swag内部的package信息，所有复制出来使用.不修改逻辑 ////////////////////////////////
// 【改造4】 添加字段
type RouteProperties struct {
	HTTPMethod  string
	Path        string
	PackageName string
	PackagePath string
	MethodName  string
	Receiver    string
	Limit       Limit
}

type Limit struct {
	Name string
	Num  string
	Cap  string
}

type PackagesDefinitions struct {
	files             map[*ast.File]*swag.AstFileInfo
	packages          map[string]*swag.PackageDefinitions
	uniqueDefinitions map[string]*swag.TypeSpecDef
}

// NewPackagesDefinitions create object PackagesDefinitions.
func NewPackagesDefinitions() *PackagesDefinitions {
	return &PackagesDefinitions{
		files:             make(map[*ast.File]*swag.AstFileInfo),
		packages:          make(map[string]*swag.PackageDefinitions),
		uniqueDefinitions: make(map[string]*swag.TypeSpecDef),
	}
}

// CollectAstFile collect ast.file.
func (pkgs *PackagesDefinitions) CollectAstFile(packageDir, path string, astFile *ast.File) error {
	if pkgs.files == nil {
		pkgs.files = make(map[*ast.File]*swag.AstFileInfo)
	}

	if pkgs.packages == nil {
		pkgs.packages = make(map[string]*swag.PackageDefinitions)
	}

	// return without storing the file if we lack a packageDir
	if packageDir == "" {
		return nil
	}

	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	pd, ok := pkgs.packages[packageDir]
	if ok {
		// return without storing the file if it already exists
		_, exists := pd.Files[path]
		if exists {
			return nil
		}
		pd.Files[path] = astFile
	} else {
		pkgs.packages[packageDir] = &swag.PackageDefinitions{
			Name:            astFile.Name.Name,
			Files:           map[string]*ast.File{path: astFile},
			TypeDefinitions: make(map[string]*swag.TypeSpecDef),
		}
	}

	pkgs.files[astFile] = &swag.AstFileInfo{
		File:        astFile,
		Path:        path,
		PackagePath: packageDir,
	}

	return nil
}
