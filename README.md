# btcgo v2.0

Este código é um software para tentar encontrar chaves privadas de Bitcoin que correspondem a endereços específicos. Ele utiliza um método de busca conhecido como "busca em blocos", onde divide o espaço de chaves privadas em blocos menores e os vasculha em busca de correspondências.

Método de Busca em Blocos:

Carregamento de Dados: O código carrega uma lista de endereços de Bitcoin a partir do arquivo "wallets.json" e uma lista de faixas de chaves privadas (blocos) a partir do arquivo "ranges.json". Cada bloco contém um intervalo de chaves privadas, definido por valores mínimo e máximo.

Seleção da Faixa: O usuário seleciona a faixa de busca (bloco) desejada.

Divisão do Bloco: O bloco selecionado é dividido em blocos ainda menores, cujo tamanho pode ser ajustado pelo usuário (tamanho do bloco).

Paralelização da Busca: O código utiliza várias threads (goroutines) para executar a busca em paralelo, aumentando a velocidade.

Busca em Blocos Aleatórios: Cada thread recebe um bloco aleatório dentro da faixa selecionada.

Varredura do Bloco: A thread percorre todas as chaves privadas do bloco, convertendo-as para o formato WIF e gerando o endereço Bitcoin correspondente.

Verificação de Correspondência: Em cada etapa, o endereço gerado é comparado com os endereços da lista "wallets.json". Se houver correspondência, a chave privada (WIF) e o endereço são exibidos e salvos em um arquivo.

Progresso da Busca: O programa salva o progresso da busca, registrando os blocos já verificados, para que possa ser retomada do ponto anterior em caso de interrupção.

Visualização do Progresso: O código exibe periodicamente o número de chaves verificadas e a taxa de verificação (chaves por segundo).

Término da Busca: A busca continua até que todas as chaves da faixa selecionada sejam verificadas, até que o usuário interrompa manualmente, ou até que uma chave correspondente seja encontrada.

Observações:

Probabilidade: Encontrar uma chave privada correspondente a um endereço aleatório é extremamente improvável, como encontrar uma agulha em um palheiro.
Recursos Computacionais: O processo exige muitos recursos computacionais e pode levar muito tempo, dependendo do tamanho da faixa de busca.
Segurança: Armazene a lista de endereços "wallets.json" e o arquivo de progresso em locais seguros.
Importante: Encontrar uma chave privada que não lhe pertence é ilegal e pode ter consequências graves. Utilize este código apenas para fins educacionais ou para buscar chaves privadas que você sabe que são suas.
