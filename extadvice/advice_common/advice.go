package advice_common

import (
	"embed"
	"github.com/rs/zerolog/log"
)

func ReadAdviceFile(fs embed.FS, fileName string) string {
	fileContent, err := fs.ReadFile(fileName)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to read file: %s", fileName)
	}
	return string(fileContent)
}
