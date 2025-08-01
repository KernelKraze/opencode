#!/usr/bin/env bun

import { $ } from "bun"

import pkg from "../package.json"

const dry = process.argv.includes("--dry")
const snapshot = process.argv.includes("--snapshot")
const skipNpm = process.env["SKIP_NPM_PUBLISH"] === "true" || !process.env["NPM_CONFIG_TOKEN"]

// 动态获取当前仓库信息
async function getRepoInfo() {
  const remoteUrl = await $`git config --get remote.origin.url`.text().then((x) => x.trim())

  // 支持 SSH 和 HTTPS 格式
  // git@github.com:KernelKraze/opencode.git -> KernelKraze/opencode
  // https://github.com/KernelKraze/opencode.git -> KernelKraze/opencode
  const match = remoteUrl.match(/github\.com[:/](.+?)(?:\.git)?$/)
  if (!match) {
    console.error("无法解析 GitHub 仓库信息:", remoteUrl)
    process.exit(1)
  }

  const [owner, repo] = match[1].split("/")
  return { owner, repo, fullName: match[1] }
}

const repoInfo = await getRepoInfo()
console.log(`检测到仓库: ${repoInfo.fullName}`)

// 获取项目根目录的绝对路径
const projectRoot = process.cwd().includes("packages/opencode")
  ? process.cwd().split("packages/opencode")[0]
  : process.cwd()
const tuiPath = `${projectRoot}/packages/tui`
const outputBasePath = `${projectRoot}/packages/opencode/dist`

const version = snapshot
  ? `0.0.0-${new Date().toISOString().slice(0, 16).replace(/[-:T]/g, "")}`
  : await $`git describe --tags --abbrev=0`
      .text()
      .then((x) => x.substring(1).trim())
      .catch(() => {
        console.error("tag not found")
        process.exit(1)
      })

console.log(`publishing ${version}`)

const GOARCH: Record<string, string> = {
  arm64: "arm64",
  x64: "amd64",
  "x64-baseline": "amd64",
}

const targets = [
  ["linux", "arm64"],
  ["linux", "x64"],
  ["linux", "x64-baseline"],
  ["darwin", "x64"],
  ["darwin", "arm64"],
  ["windows", "x64"],
]

await $`rm -rf dist`

const optionalDependencies: Record<string, string> = {}
const npmTag = snapshot ? "snapshot" : "latest"
for (const [os, arch] of targets) {
  console.log(`building ${os}-${arch}`)
  const name = `${pkg.name}-${os}-${arch}`
  await $`mkdir -p dist/${name}/bin`
  await $`CGO_ENABLED=0 GOOS=${os} GOARCH=${GOARCH[arch]} go build -ldflags="-s -w -X main.Version=${version}" -o ${outputBasePath}/${name}/bin/tui ./cmd/opencode/main.go`.cwd(
    tuiPath,
  )
  await $`bun build --define OPENCODE_VERSION="'${version}'" --compile --minify --target=bun-${os}-${arch} --outfile=dist/${name}/bin/opencode ./src/index.ts ./dist/${name}/bin/tui`
  await $`rm -rf ./dist/${name}/bin/tui`
  await Bun.file(`dist/${name}/package.json`).write(
    JSON.stringify(
      {
        name,
        version,
        os: [os === "windows" ? "win32" : os],
        cpu: [arch],
      },
      null,
      2,
    ),
  )
  if (!dry && !skipNpm) {
    try {
      await $`cd dist/${name} && bun publish --access public --tag ${npmTag}`
    } catch (err) {
      console.warn(`跳过NPM发布 ${name}: ${err instanceof Error ? err.message : String(err)}`)
    }
  }
  optionalDependencies[name] = version
}

await $`mkdir -p ./dist/${pkg.name}`
await $`cp -r ./bin ./dist/${pkg.name}/bin`
await $`cp ./script/postinstall.mjs ./dist/${pkg.name}/postinstall.mjs`
await Bun.file(`./dist/${pkg.name}/package.json`).write(
  JSON.stringify(
    {
      name: pkg.name + "-ai",
      bin: {
        [pkg.name]: `./bin/${pkg.name}`,
      },
      scripts: {
        postinstall: "node ./postinstall.mjs",
      },
      version,
      optionalDependencies,
    },
    null,
    2,
  ),
)
if (!dry && !skipNpm) {
  try {
    await $`cd ./dist/${pkg.name} && bun publish --access public --tag ${npmTag}`
  } catch (err) {
    console.warn(`跳过主包NPM发布: ${err instanceof Error ? err.message : String(err)}`)
  }
}

if (!snapshot) {
  // Github Release
  for (const key of Object.keys(optionalDependencies)) {
    await $`cd dist/${key}/bin && zip -r ../../${key}.zip *`
  }

  // GitHub Release 相关 API 调用现在使用动态仓库信息
  // 处理首次发布场景：当仓库还没有任何release时，API返回404
  const previous = await fetch(`https://api.github.com/repos/${repoInfo.fullName}/releases/latest`)
    .then((res) => {
      if (!res.ok) {
        if (res.status === 404) {
          console.log("这是首次发布，没有找到之前的release")
          return null
        }
        throw new Error(res.statusText)
      }
      return res.json()
    })
    .then((data) => data?.tag_name || null)
    .catch((err) => {
      console.log("获取最新release失败，可能是首次发布:", err.message)
      return null
    })

  let commits = []
  if (previous) {
    console.log("finding commits between", previous, "and", "HEAD")
    commits = await fetch(`https://api.github.com/repos/${repoInfo.fullName}/compare/${previous}...HEAD`)
      .then((res) => res.json())
      .then((data) => data.commits || [])
      .catch((err) => {
        console.log("获取commit对比失败:", err.message)
        return []
      })
  } else {
    console.log("首次发布，获取最近的commits作为release notes")
    commits = await fetch(`https://api.github.com/repos/${repoInfo.fullName}/commits?per_page=10`)
      .then((res) => res.json())
      .then((data) => data || [])
      .catch((err) => {
        console.log("获取最近commits失败:", err.message)
        return []
      })
  }

  const raw = commits.map((commit: any) => `- ${commit.commit.message.split("\n").join(" ")}`)
  console.log(raw)

  const notes =
    raw
      .filter((x: string) => {
        const lower = x.toLowerCase()
        return (
          !lower.includes("ignore:") &&
          !lower.includes("chore:") &&
          !lower.includes("ci:") &&
          !lower.includes("wip:") &&
          !lower.includes("docs:") &&
          !lower.includes("doc:")
        )
      })
      .join("\n") || "No notable changes"

  if (!dry) await $`gh release create v${version} --title "v${version}" --notes ${notes} ./dist/*.zip`

  // Calculate SHA values
  const arm64Sha = await $`sha256sum ./dist/opencode-linux-arm64.zip | cut -d' ' -f1`.text().then((x) => x.trim())
  const x64Sha = await $`sha256sum ./dist/opencode-linux-x64.zip | cut -d' ' -f1`.text().then((x) => x.trim())
  const macX64Sha = await $`sha256sum ./dist/opencode-darwin-x64.zip | cut -d' ' -f1`.text().then((x) => x.trim())
  const macArm64Sha = await $`sha256sum ./dist/opencode-darwin-arm64.zip | cut -d' ' -f1`.text().then((x) => x.trim())

  // AUR package
  const pkgbuild = [
    "# Maintainer: dax",
    "# Maintainer: adam",
    "",
    "pkgname='${pkg}'",
    `pkgver=${version.split("-")[0]}`,
    "options=('!debug' '!strip')",
    "pkgrel=1",
    "pkgdesc='The AI coding agent built for the terminal.'",
    `url='https://github.com/${repoInfo.fullName}'`,
    "arch=('aarch64' 'x86_64')",
    "license=('MIT')",
    "provides=('opencode')",
    "conflicts=('opencode')",
    "depends=('fzf' 'ripgrep')",
    "",
    `source_aarch64=("\${pkgname}_\${pkgver}_aarch64.zip::https://github.com/${repoInfo.fullName}/releases/download/v${version}/opencode-linux-arm64.zip")`,
    `sha256sums_aarch64=('${arm64Sha}')`,
    "",
    `source_x86_64=("\${pkgname}_\${pkgver}_x86_64.zip::https://github.com/${repoInfo.fullName}/releases/download/v${version}/opencode-linux-x64.zip")`,
    `sha256sums_x86_64=('${x64Sha}')`,
    "",
    "package() {",
    '  install -Dm755 ./opencode "${pkgdir}/usr/bin/opencode"',
    "}",
    "",
  ].join("\n")

  for (const pkg of ["opencode", "opencode-bin"]) {
    await $`rm -rf ./dist/aur-${pkg}`
    await $`git clone ssh://aur@aur.archlinux.org/${pkg}.git ./dist/aur-${pkg}`
    await $`cd ./dist/aur-${pkg} && git checkout master`
    await Bun.file(`./dist/aur-${pkg}/PKGBUILD`).write(pkgbuild.replace("${pkg}", pkg))
    await $`cd ./dist/aur-${pkg} && makepkg --printsrcinfo > .SRCINFO`
    await $`cd ./dist/aur-${pkg} && git add PKGBUILD .SRCINFO`
    await $`cd ./dist/aur-${pkg} && git commit -m "Update to v${version}"`
    if (!dry) await $`cd ./dist/aur-${pkg} && git push`
  }

  // Homebrew formula
  const homebrewFormula = [
    "# typed: false",
    "# frozen_string_literal: true",
    "",
    "# This file was generated by GoReleaser. DO NOT EDIT.",
    "class Opencode < Formula",
    `  desc "The AI coding agent built for the terminal."`,
    `  homepage "https://github.com/${repoInfo.fullName}"`,
    `  version "${version.split("-")[0]}"`,
    "",
    "  on_macos do",
    "    if Hardware::CPU.intel?",
    `      url "https://github.com/${repoInfo.fullName}/releases/download/v${version}/opencode-darwin-x64.zip"`,
    `      sha256 "${macX64Sha}"`,
    "",
    "      def install",
    '        bin.install "opencode"',
    "      end",
    "    end",
    "    if Hardware::CPU.arm?",
    `      url "https://github.com/${repoInfo.fullName}/releases/download/v${version}/opencode-darwin-arm64.zip"`,
    `      sha256 "${macArm64Sha}"`,
    "",
    "      def install",
    '        bin.install "opencode"',
    "      end",
    "    end",
    "  end",
    "",
    "  on_linux do",
    "    if Hardware::CPU.intel? and Hardware::CPU.is_64_bit?",
    `      url "https://github.com/${repoInfo.fullName}/releases/download/v${version}/opencode-linux-x64.zip"`,
    `      sha256 "${x64Sha}"`,
    "      def install",
    '        bin.install "opencode"',
    "      end",
    "    end",
    "    if Hardware::CPU.arm? and Hardware::CPU.is_64_bit?",
    `      url "https://github.com/${repoInfo.fullName}/releases/download/v${version}/opencode-linux-arm64.zip"`,
    `      sha256 "${arm64Sha}"`,
    "      def install",
    '        bin.install "opencode"',
    "      end",
    "    end",
    "  end",
    "end",
    "",
    "",
  ].join("\n")

  await $`rm -rf ./dist/homebrew-tap`
  await $`git clone https://${process.env["GITHUB_TOKEN"]}@github.com/sst/homebrew-tap.git ./dist/homebrew-tap`
  await Bun.file("./dist/homebrew-tap/opencode.rb").write(homebrewFormula)
  await $`cd ./dist/homebrew-tap && git add opencode.rb`
  await $`cd ./dist/homebrew-tap && git commit -m "Update to v${version}"`
  if (!dry) await $`cd ./dist/homebrew-tap && git push`
}
