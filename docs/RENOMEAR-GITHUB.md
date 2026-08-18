# Renomear no GitHub

O CLI `gh` não estava autenticado neste ambiente. Depois de `gh auth login`:

```bash
gh repo rename oficina-api --yes --repo RenanAlvesBCC/todolist-api
gh repo rename oficina-gerente-web --yes --repo RenanAlvesBCC/oficina-web
gh repo rename oficina-mecanico-ios --yes --repo RenanAlvesBCC/todolist-ios

cd ~/Documents/projetos/oficina-api && git remote set-url origin git@github.com:RenanAlvesBCC/oficina-api.git
cd ~/Documents/projetos/oficina-gerente-web && git remote set-url origin git@github.com:RenanAlvesBCC/oficina-gerente-web.git
cd ~/Documents/projetos/todolist-ios && git remote set-url origin git@github.com:RenanAlvesBCC/oficina-mecanico-ios.git
mv ~/Documents/projetos/todolist-ios ~/Documents/projetos/oficina-mecanico-ios

cd ~/Documents/projetos/oficina-mecanico-android
gh repo create RenanAlvesBCC/oficina-mecanico-android --private --source=. --remote=origin --push
```
