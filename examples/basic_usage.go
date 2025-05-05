package main

import (
	"fmt"
	"time"

	"github.com/iamviniciuss/observability_go/pkg/timer"
)

func main() {
	// Exemplo 1: Usando TimeFunc para medir uma função simples
	resultado, duracao := timer.TimeFunc(func() string {
		// Simulando algum processamento
		time.Sleep(100 * time.Millisecond)
		return "processamento concluído"
	})
	fmt.Printf("Resultado: %s, Duração: %s\n", resultado, duracao)

	// Exemplo 2: Usando WithTiming como decorator com callback
	funcaoMedida := timer.WithTiming(
		func() int {
			// Simulando outro processamento
			time.Sleep(200 * time.Millisecond)
			return 42
		},
		func(d time.Duration) {
			fmt.Printf("A função demorou %s para executar\n", d)
		},
	)
	resultado2 := funcaoMedida()
	fmt.Printf("Resultado da função medida: %d\n", resultado2)

	// Exemplo 3: Usando MeasureTime para medir blocos de código
	defer timer.MeasureTime("operação completa")()

	// Simulando diferentes partes de uma operação
	{
		defer timer.MeasureTime("parte 1")()
		time.Sleep(50 * time.Millisecond)
	}

	{
		defer timer.MeasureTime("parte 2")()
		time.Sleep(75 * time.Millisecond)
	}
}
