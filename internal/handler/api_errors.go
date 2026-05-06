package handler

import (
	"errors"
	"net/http"

	"github.com/morinonusi421/cupid/internal/model"
	"github.com/morinonusi421/cupid/internal/service"
	"github.com/morinonusi421/cupid/pkg/httputil"
)

// simpleAPIErrors は「コードのみ返せば足りるエラー」の sentinel error → 公開コード/ステータスの対応表。
//
// このマップに載っているエラーは tryWriteSimpleAPIError で機械的に処理される。
// ここに無いエラーは「予期しない内部エラー」として 500 にフォールバックする。
//
// マップに載せる条件:
//   - クライアントに返してよい安定した識別子（API契約）を持つ
//   - 追加のレスポンスフィールド（partner_name 等）が不要
//
// 追加データが必要なエラー（user_not_found の user_liff_url、matched_user_exists の partner_name 等）は
// このマップではなく、handler 内で明示的に分岐させる。
var simpleAPIErrors = map[error]struct {
	code   string
	status int
}{
	service.ErrCannotRegisterYourself: {"cannot_register_yourself", http.StatusBadRequest},
	service.ErrDuplicateUser:          {"duplicate_user", http.StatusConflict},
	model.ErrNameInvalidLength:        {"name_invalid_length", http.StatusBadRequest},
	model.ErrNameInvalidFormat:        {"name_invalid_format", http.StatusBadRequest},
}

// tryWriteSimpleAPIError は err が simpleAPIErrors に登録された sentinel と一致する場合、
// 対応する公開コードを JSON で書き出して true を返す。一致しなければ false。
func tryWriteSimpleAPIError(w http.ResponseWriter, err error) bool {
	for sentinel, info := range simpleAPIErrors {
		if errors.Is(err, sentinel) {
			httputil.WriteJSONError(w, info.status, map[string]string{"error": info.code})
			return true
		}
	}
	return false
}
