require('hardhat');

async function main() {
  const [deployer] = await ethers.getSigners();

  console.log("Deploying contracts with the account:", deployer.address);

  console.log("Account balance:", (await deployer.getBalance()).toString());

  const Mirror = await ethers.getContractFactory("Mirror");
  const mirror = await Mirror.deploy("0x3c84B6C98FBeB813e05a7A7813F0442883450B1F");
  console.log("Mirror address:", mirror.address);
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
