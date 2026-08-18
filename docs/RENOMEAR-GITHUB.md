# Renomear no GitHub

O app Android já está em `RenanAlvesBCC/oficina-mecanico-android`.

`gh` precisa estar autenticado (`gh auth login`) para os renames restantes:

```bash
gh repo rename oficina-api --yes --repo RenanAlvesBCC/todolist-api
gh repo rename oficina-gerente-web --yes --repo RenanAlvesBCC/oficina-web
gh repo rename oficina-mecanico-ios --yes --repo RenanAlvesBCC/todolist-ios

cd ~/Documents/projetos/oficina-api && git remote set-url origin git@github.com:RenanAlvesBCC/oficina-api.git
cd ~/Documents/projetos/oficina-gerente-web && git remote set-url origin git@github.com:RenanAlvesBCC/oficina-gerente-web.git
cd ~/Documents/projetos/todolist-ios && git remote set-url origin git@github.com:RenanAlvesBCC/oficina-mecanico-ios.git
mv ~/Documents/projetos/todolist-ios ~/Documents/projetos/oficina-mecanico-ios
```

Identidade visível (bundle, display name, strings da UI) já usa Oficina. O scheme/target `TodoList` no Xcode pode mudar depois do rename remoto.
