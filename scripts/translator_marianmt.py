import argparse
from transformers import MarianMTModel, MarianTokenizer


def main():
    parser = argparse.ArgumentParser(
        description="Traducao de textos usando MarianMT")
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument("--text", type=str, help="Texto unico para traduzir")
    group.add_argument("--input", type=str,
                        help="Arquivo com textos (um por linha)")
    parser.add_argument("--output", type=str, help="Arquivo de saida")
    parser.add_argument("--target-lang", type=str, default="pt_BR")
    parser.add_argument(
        "--model", type=str, default="Helsinki-NLP/opus-mt-en-ROMANCE",
        help="Nome do modelo de traducao")

    args = parser.parse_args()

    if args.text:
        texts = [args.text.strip('"')]
    else:
        with open(args.input, "r", encoding="utf-8") as f:
            texts = [line.strip() for line in f.readlines() if line.strip()]

    tokenizer = MarianTokenizer.from_pretrained(args.model)
    model = MarianMTModel.from_pretrained(args.model)

    src_texts = [f">>{args.target_lang}<< {text}" for text in texts]
    translated = model.generate(
        **tokenizer(src_texts, return_tensors="pt", padding=True, max_length=512))

    results = []
    for t in translated:
        results.append(tokenizer.decode(t, skip_special_tokens=True))

    output = "\n".join(results)

    if args.output:
        with open(args.output, "w", encoding="utf-8") as f:
            f.write(output)
    else:
        print(output)


if __name__ == "__main__":
    main()
