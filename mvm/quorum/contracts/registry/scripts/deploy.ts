import { ethers } from "hardhat";

async function main() {
  const [deployer] = await ethers.getSigners();

  console.log("Deploying contracts with the account:", deployer.address);

  console.log("Account balance:", (await deployer.getBalance()).toString());

  const Storage = await ethers.getContractFactory("Storage");
  const storage = await Storage.deploy();
  console.log("Storage address:", storage.address);

  //const BLS = await ethers.getContractFactory("BLS");
  //const bls = await BLS.deploy();
  //console.log("BLS address:", bls.address);

  const Registry = await ethers.getContractFactory("Registry", {
    //libraries: {
      //BLS: bls.address
    //}
  });
  //const registry = await Registry.deploy("0x1b0b73f760f5a1fc2d3b14b18a1fb5f7d8e93366ac283423c7b6413dd869bf1300a4398f2222cca7c31cd56b4557249bf9f3c538b30b06e3a9a4c830a9b48feb25fc87924ed7906607d59b6e9555230e73b46378252923ac719a328c7235cb03030e7999862a645112eefee572b4f930a8c66b4141d0f8d76558364ed539c03e", 0x148e696ff1db4472a907ceea50c5cfde);
  const registry = await Registry.deploy("0x1b0b73f760f5a1fc2d3b14b18a1fb5f7d8e93366ac283423c7b6413dd869bf1300a4398f2222cca7c31cd56b4557249bf9f3c538b30b06e3a9a4c830a9b48feb25fc87924ed7906607d59b6e9555230e73b46378252923ac719a328c7235cb03030e7999862a645112eefee572b4f930a8c66b4141d0f8d76558364ed539c03e", "0xc1eff8f5b129395d8537b54a5a525f85");

  console.log("Registry address:", registry.address);
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
