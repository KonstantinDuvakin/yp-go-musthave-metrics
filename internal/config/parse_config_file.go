package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ParseFile заполняет c значениями из JSON-файла конфигурации.
//
// Путь к файлу берётся из переменной окружения CONFIG, а если она не
// задана — из флага -c или его полной формы -config (см. [configFlagValue]).
// Если путь не указан нигде, файл не читается и c не меняется.
//
// Вызывать нужно до flag.Parse(): поля, которых нет в файле, сохраняют
// значения по умолчанию, а флаги и переменные окружения затем
// перекрывают значения из файла.
func ParseFile(c json.Unmarshaler) error {
	args := os.Args[1:]
	configPath, err := configFlagValue(args)
	if err != nil {
		return fmt.Errorf("error parse config flag: %w", err)
	}

	if envConfPath, ok := os.LookupEnv("CONFIG"); ok {
		configPath = envConfPath
	}

	if configPath == "" {
		return nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("error read config file: %w", err)
	}

	err = json.Unmarshal(data, c)
	if err != nil {
		return fmt.Errorf("error unmarshal config data: %w", err)
	}

	return nil
}

// configFlagValue ищет в args (аргументы без имени программы) значение
// флага пути к конфигу. Флаг имеет два равноправных имени, -c и -config,
// и принимается в тех же формах, что понимает пакет flag: "-c path",
// "-c=path", "--c path", "--c=path" и так же для -config.
//
// Если флаг указан несколько раз (под любым из имён), побеждает последний,
// как в flag.Parse. Если флага нет, возвращает пустую строку. Если после
// флага без "=" нет значения, возвращает ошибку.
func configFlagValue(args []string) (string, error) {
	var value string
	for i, arg := range args {
		isEnoughLen := i+1 < len(args)

		flagName, v, isFound := strings.Cut(arg, "=")

		if flagName == "-c" || flagName == "--c" || flagName == "-config" || flagName == "--config" {
			if !isFound {
				if !isEnoughLen {
					return "", fmt.Errorf("not enough arguments for flag %s", flagName)
				}

				value = args[i+1]
				continue
			}

			value = v
		}
	}

	return value, nil
}
