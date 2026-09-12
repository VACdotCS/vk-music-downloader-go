package scenarios

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/pterm/pterm"
	"vk-music-downloader-go/internal/config"
)

func GetUnlimitedTokenScenario(myConfig *config.Config) error {
	fmt.Println("Откройте ссылку: https://oauth.vk.com/authorize?client_id=6463690&scope=1073737727&redirect_uri=https://oauth.vk.com/blank.html&display=page&response_type=token&revoke=1")
	fmt.Println("Разрешите доступ. \nИз выданных прав используются только аудиозаписи. \nИсходный код программы открыт.")
	fmt.Println("О способе узнал тут: https://vkhost.github.io/")

	var tokenLink string
	prompt := &survey.Input{Message: "Вставьте данные из адресной строки:"}
	if err := survey.AskOne(prompt, &tokenLink); err != nil {
		return err
	}

	parts := strings.Split(tokenLink, "#")
	if len(parts) < 2 {
		pterm.Error.Println("Неверный формат ссылки")
		return fmt.Errorf("invalid format")
	}

	res := make(map[string]string)
	params := strings.Split(parts[1], "&")
	for _, p := range params {
		kv := strings.Split(p, "=")
		if len(kv) == 2 {
			res[kv[0]] = kv[1]
		}
	}

	accessToken := res["access_token"]
	userIDStr := res["user_id"]
	if accessToken == "" || userIDStr == "" {
		pterm.Error.Println("Не удалось найти токен или user_id")
		return fmt.Errorf("missing data")
	}

	userID, _ := strconv.Atoi(userIDStr)

	if myConfig.Token == nil {
		myConfig.Token = &config.Token{}
	}
	myConfig.Token.AccessToken = accessToken
	myConfig.Token.UserID = userID
	myConfig.Token.Expires = math.MaxInt64 // Бесконечный токен

	config.SaveConfig(myConfig)
	pterm.Success.Println("Бесконечный токен успешно сохранен!")
	return nil
}
