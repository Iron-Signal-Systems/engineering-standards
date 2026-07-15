# Create and Publish the Repository

The GitHub repository must be created once in the Iron Signal Systems
organization because repository creation is an organization-level action.

## 1. Create the empty repository

In GitHub:

1. Open the `Iron-Signal-Systems` organization.
2. Select **New repository**.
3. Repository name: `engineering-standards`.
4. Description: `Iron Signal Systems engineering, repository assurance, secure-development, validation, acceptance, release, and operational evidence standards.`
5. Choose public or private according to the company publishing decision.
6. Do not initialize it with a README, license, or `.gitignore`.
7. Create the repository.

The canonical location will be:

```text
https://github.com/Iron-Signal-Systems/engineering-standards
```

## 2. Extract the package and initialize Git

```bash
cd ~/Dev/projects
unzip engineering-standards-v1.0.0.zip
mv engineering-standards engineering-standards
cd engineering-standards

git init -b dev
git add .
git commit -m "establish Iron Signal engineering standards v1"
git remote add origin git@github.com:Iron-Signal-Systems/engineering-standards.git
git push -u origin dev
```

Create `main` at the same accepted initial commit:

```bash
git branch main
git push -u origin main
```

Set `dev` as the default branch in GitHub, then return to `dev` for all work.

## 3. Validate the published repository

```bash
python3 tools/isras/validate_policy.py --repo-root .
python3 tools/isras/validate_fresh_clone.py --repo-root .
```

## 4. Initial protection sequence

Begin in observation mode:

- workflows enabled;
- no required checks yet;
- no required second reviewer while only one qualified maintainer exists.

After stable runs:

- require pull requests into `dev`;
- require policy and portable checks;
- block force pushes and branch deletion;
- protect `main`;
- protect `v*`, `phase-*`, and `release-*` tags;
- document one emergency administrator bypass.
