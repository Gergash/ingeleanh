const hre = require("hardhat");

async function main() {
  const C2Registry = await hre.ethers.getContractFactory("C2Registry");
  const registry = await C2Registry.deploy();
  await registry.waitForDeployment();
  const address = await registry.getAddress();
  console.log("C2Registry deployed to:", address);
  console.log("Set C2_REGISTRY_ADDRESS=" + address);
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
