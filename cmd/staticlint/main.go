// Package main предоставляет статический анализатор для проверки использования
// вызова os.Exit в функции main пакета main, а также интеграцию с другими
// стандартными и пользовательскими анализаторами с помощью multichecker.
package main

import (
	"github.com/timakin/bodyclose/passes/bodyclose"
	"go/ast"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/appends"
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/atomicalign"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/ctrlflow"
	"golang.org/x/tools/go/analysis/passes/deepequalerrors"
	"golang.org/x/tools/go/analysis/passes/defers"
	"golang.org/x/tools/go/analysis/passes/directive"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/fieldalignment"
	"golang.org/x/tools/go/analysis/passes/findcall"
	"golang.org/x/tools/go/analysis/passes/framepointer"
	"golang.org/x/tools/go/analysis/passes/httpmux"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/pkgfact"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/reflectvaluecompare"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sigchanyzer"
	"golang.org/x/tools/go/analysis/passes/slog"
	"golang.org/x/tools/go/analysis/passes/sortslice"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stdversion"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/testinggoroutine"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/timeformat"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"golang.org/x/tools/go/analysis/passes/unusedwrite"
	"golang.org/x/tools/go/analysis/passes/usesgenerics"
	"honnef.co/go/tools/analysis/facts/nilness"
	"honnef.co/go/tools/staticcheck"
)

// ExitCheckAnalyzer - анализатор, который проверяет использование os.Exit в функции main пакета main.
// Он генерирует предупреждение, если обнаруживает прямой вызов os.Exit в функции main.
var ExitCheckAnalyzer = &analysis.Analyzer{
	Name: "exitcheck",
	Doc:  "checks that os.Exit is not called directly in the main function",
	Run:  run,
}

// run - функция, выполняющая анализ. Она проверяет, что вызов os.Exit не используется
// в функции main пакета main.
func run(pass *analysis.Pass) (interface{}, error) {
	// Проверяем, что анализируем пакет "main"
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	// Проходим по всем объявлениям в файле
	for _, file := range pass.Files {
		// Ищем функции main
		ast.Inspect(file, func(n ast.Node) bool {
			// Если нашли функцию main, проверяем на вызов os.Exit
			funcDecl, ok := n.(*ast.FuncDecl)
			if ok && funcDecl.Name.Name == "main" {
				// Проходим по телу функции
				ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
					// Ищем вызовы os.Exit
					callExpr, ok := n.(*ast.CallExpr)
					if ok {
						// Проверяем, что это вызов os.Exit
						switch fun := callExpr.Fun.(type) {
						case *ast.SelectorExpr:
							if pkg, ok := fun.X.(*ast.Ident); ok && pkg.Name == "os" && fun.Sel.Name == "Exit" {
								pass.Reportf(callExpr.Pos(), "direct call to os.Exit in main function")
							}
						}
					}
					return true
				})
			}
			return true
		})
	}

	return nil, nil
}

// main - основной метод, который запускает multichecker с набором анализаторов.
// Он также добавляет собственный анализатор для проверки вызова os.Exit в функции main.
func main() {
	// Список анализаторов для multichecker
	mychecks := []*analysis.Analyzer{
		// Стандартные анализаторы из пакета golang.org/x/tools
		appends.Analyzer,
		asmdecl.Analyzer,
		assign.Analyzer,
		atomic.Analyzer,
		atomicalign.Analyzer,
		bools.Analyzer,
		buildssa.Analyzer,
		buildtag.Analyzer,
		cgocall.Analyzer,
		composite.Analyzer,
		copylock.Analyzer,
		ctrlflow.Analyzer,
		deepequalerrors.Analyzer,
		defers.Analyzer,
		directive.Analyzer,
		errorsas.Analyzer,
		fieldalignment.Analyzer,
		findcall.Analyzer,
		framepointer.Analyzer,
		httpmux.Analyzer,
		httpresponse.Analyzer,
		ifaceassert.Analyzer,
		inspect.Analyzer,
		loopclosure.Analyzer,
		lostcancel.Analyzer,
		nilfunc.Analyzer,
		nilness.Analysis,
		pkgfact.Analyzer,
		printf.Analyzer,
		reflectvaluecompare.Analyzer,
		shadow.Analyzer,
		shift.Analyzer,
		sigchanyzer.Analyzer,
		slog.Analyzer,
		sortslice.Analyzer,
		stdmethods.Analyzer,
		stdversion.Analyzer,
		stringintconv.Analyzer,
		structtag.Analyzer,
		testinggoroutine.Analyzer,
		tests.Analyzer,
		timeformat.Analyzer,
		unmarshal.Analyzer,
		unreachable.Analyzer,
		unsafeptr.Analyzer,
		unusedresult.Analyzer,
		unusedwrite.Analyzer,
		usesgenerics.Analyzer,
		// Добавляем наш собственный анализатор
		ExitCheckAnalyzer,
	}

	// Добавляем анализаторы из staticcheck
	for _, a := range staticcheck.Analyzers {
		if a.Analyzer != nil && a.Analyzer.Name[:2] == "SA" {
			mychecks = append(mychecks, a.Analyzer)
		}
		if a.Analyzer.Name == "ST1000" {
			mychecks = append(mychecks, a.Analyzer)
		}
	}

	// Добавляем анализатор bodyclose
	mychecks = append(mychecks, bodyclose.Analyzer)

	// Запуск multichecker с набором анализаторов
	multichecker.Main(mychecks...)
}
