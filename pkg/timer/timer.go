// Package timer fornece funcionalidades para medir o tempo de execução de funções
package timer

import (
	"fmt"
	"time"
)

// TimeFunc executa a função fornecida e retorna o resultado e a duração da execução
// Para funções sem retorno, use TimeFunc(func() { minhaFuncao() })
func TimeFunc[T any](fn func() T) (result T, duration time.Duration) {
	start := time.Now()
	result = fn()
	duration = time.Since(start)
	return result, duration
}

// WithTiming retorna uma nova função que executa a função original e retorna o resultado,
// mas também mede o tempo de execução usando a função de callback fornecida
func WithTiming[T any](fn func() T, callback func(time.Duration)) func() T {
	return func() T {
		start := time.Now()
		result := fn()
		duration := time.Since(start)

		if callback != nil {
			callback(duration)
		}

		return result
	}
}

// MeasureTime é uma utility para medir o tempo de execução de um bloco de código
// Exemplo de uso:
//
//	defer timer.MeasureTime("operacao")()
func MeasureTime(operation string) func() {
	start := time.Now()
	fmt.Printf("⏱️ Iniciando medição: %s\n", operation)
	return func() {
		duration := time.Since(start)
		// Por padrão, apenas imprime no log
		// Em versões futuras, isso pode ser integrado com OpenTelemetry
		// ou outros sistemas de observabilidade
		println(operation+":", duration.String())
	}
}
