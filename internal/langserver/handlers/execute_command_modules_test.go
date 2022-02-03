package handlers

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/creachadair/jrpc2/code"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/langserver"
	"github.com/hashicorp/terraform-ls/internal/langserver/cmd"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/stretchr/testify/mock"
)

func TestLangServer_workspaceExecuteCommand_modules_argumentError(t *testing.T) {
	tmpDir := TempDir(t)
	testFileURI := fmt.Sprintf("%s/main.tf", tmpDir.URI)
	InitPluginCache(t, tmpDir.Path())

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		TerraformCalls: &exec.TerraformMockCalls{
			PerWorkDir: map[string][]*mock.Call{
				tmpDir.Path(): validTfMockCalls(),
			},
		},
	}))
	stop := ls.Start(t)
	defer stop()

	ls.Call(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: fmt.Sprintf(`{
	    "capabilities": {},
	    "rootUri": %q,
		"processId": 12345
	}`, tmpDir.URI)})
	ls.Notify(t, &langserver.CallRequest{
		Method:    "initialized",
		ReqParams: "{}",
	})
	ls.Call(t, &langserver.CallRequest{
		Method: "textDocument/didOpen",
		ReqParams: fmt.Sprintf(`{
		"textDocument": {
			"version": 0,
			"languageId": "terraform",
			"text": "provider \"github\" {}",
			"uri": %q
		}
	}`, testFileURI)})

	ls.CallAndExpectError(t, &langserver.CallRequest{
		Method: "workspace/executeCommand",
		ReqParams: fmt.Sprintf(`{
		"command": %q
	}`, cmd.Name("rootmodules"))}, code.InvalidParams.Err())
}

func TestLangServer_workspaceExecuteCommand_modules_basic(t *testing.T) {
	tmpDir := TempDir(t)
	testFileURI := fmt.Sprintf("%s/main.tf", tmpDir.URI)
	InitPluginCache(t, tmpDir.Path())

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		TerraformCalls: &exec.TerraformMockCalls{
			PerWorkDir: map[string][]*mock.Call{
				tmpDir.Path(): validTfMockCalls(),
			},
		},
	}))
	stop := ls.Start(t)
	defer stop()

	ls.Call(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: fmt.Sprintf(`{
	    "capabilities": {},
	    "rootUri": %q,
		"processId": 12345
	}`, tmpDir.URI)})
	ls.Notify(t, &langserver.CallRequest{
		Method:    "initialized",
		ReqParams: "{}",
	})
	ls.Call(t, &langserver.CallRequest{
		Method: "textDocument/didOpen",
		ReqParams: fmt.Sprintf(`{
		"textDocument": {
			"version": 0,
			"languageId": "terraform",
			"text": "provider \"github\" {}",
			"uri": %q
		}
	}`, testFileURI)})

	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "workspace/executeCommand",
		ReqParams: fmt.Sprintf(`{
		"command": %q,
		"arguments": ["uri=%s"] 
	}`, cmd.Name("rootmodules"), testFileURI)}, fmt.Sprintf(`{
		"jsonrpc": "2.0",
		"id": 3,
		"result": {
			"responseVersion": 0,
			"doneLoading": true,
			"rootModules": [
				{
					"uri": %q,
					"name": %q
				}
			]
		}
	}`, tmpDir.URI, t.Name()))
}

func TestLangServer_workspaceExecuteCommand_modules_multiple(t *testing.T) {
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		// The underlying API is now deprecated anyway
		// so it's not worth adapting tests for all platforms.
		// We just skip tests on Apple Silicon.
		t.Skip("deprecated API")
	}

	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}

	root := document.DirHandleFromPath(filepath.Join(testData, "main-module-multienv"))
	mod := document.DirHandleFromPath(filepath.Join(testData, "main-module-multienv", "main", "main.tf"))

	dev := document.DirHandleFromPath(filepath.Join(testData, "main-module-multienv", "env", "dev"))
	staging := document.DirHandleFromPath(filepath.Join(testData, "main-module-multienv", "env", "staging"))
	prod := document.DirHandleFromPath(filepath.Join(testData, "main-module-multienv", "env", "prod"))

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		TerraformCalls: &exec.TerraformMockCalls{
			PerWorkDir: map[string][]*mock.Call{
				dev.Path():     validTfMockCalls(),
				staging.Path(): validTfMockCalls(),
				prod.Path():    validTfMockCalls(),
			},
		},
	}))
	stop := ls.Start(t)
	defer stop()

	ls.Call(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: fmt.Sprintf(`{
	    "capabilities": {},
	    "rootUri": %q,
		"processId": 12345
	}`, root.URI)})
	ls.Notify(t, &langserver.CallRequest{
		Method:    "initialized",
		ReqParams: "{}",
	})

	// expect module definition to be associated to three modules
	// expect modules to be alphabetically sorted on uri
	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "workspace/executeCommand",
		ReqParams: fmt.Sprintf(`{
		"command": %q,
		"arguments": ["uri=%s"] 
	}`, cmd.Name("rootmodules"), mod.URI)}, fmt.Sprintf(`{
		"jsonrpc": "2.0",
		"id": 2,
		"result": {
			"responseVersion": 0,
			"doneLoading": true,
			"rootModules": [
				{
					"uri": %q,
					"name": %q
				},
				{
					"uri": %q,
					"name": %q
				},
				{
					"uri": %q,
					"name": %q
				}
			]
		}
	}`, dev.URI, filepath.Join("env", "dev"), prod.URI, filepath.Join("env", "prod"), staging.URI, filepath.Join("env", "staging")))
}
