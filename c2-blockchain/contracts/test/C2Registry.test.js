const { expect } = require("chai");
const { ethers } = require("hardhat");

describe("C2Registry", function () {
  let registry;
  let owner;
  let operator;
  let stranger;

  const pubKeyHash = ethers.keccak256(ethers.toUtf8Bytes("operator-pub"));
  const endpoint1 = ethers.keccak256(ethers.toUtf8Bytes("https://primary:8443"));
  const endpoint2 = ethers.keccak256(ethers.toUtf8Bytes("https://backup:8443"));

  beforeEach(async function () {
    [owner, operator, stranger] = await ethers.getSigners();
    const C2Registry = await ethers.getContractFactory("C2Registry");
    registry = await C2Registry.deploy();
    await registry.waitForDeployment();

    await registry.registerOperator(operator.address, pubKeyHash);
  });

  // SC-001: Solo operador activo puede updateConfig
  it("SC-001: non-operator cannot updateConfig", async function () {
    await expect(
      registry.connect(stranger).updateConfig(endpoint1, 30)
    ).to.be.revertedWithCustomError(registry, "OnlyActiveOperator");
  });

  // SC-002: version incrementa monotónicamente
  it("SC-002: getConfig version increments on each update", async function () {
    await registry.connect(operator).updateConfig(endpoint1, 30);
    let cfg = await registry.getConfig();
    expect(cfg.version).to.equal(1);

    await registry.connect(operator).updateConfig(endpoint2, 60);
    cfg = await registry.getConfig();
    expect(cfg.version).to.equal(2);

    const hist1 = await registry.configHistory(1);
    const hist2 = await registry.configHistory(2);
    expect(hist1.version).to.equal(1);
    expect(hist2.version).to.equal(2);
  });

  // SC-003: ConfigUpdated evento con campos correctos
  it("SC-003: ConfigUpdated emits correct fields", async function () {
    await expect(registry.connect(operator).updateConfig(endpoint1, 30))
      .to.emit(registry, "ConfigUpdated")
      .withArgs(1, endpoint1, 30, operator.address);
  });

  // SC-004: revokeOperator sets active=false
  it("SC-004: revokeOperator sets active=false", async function () {
    await registry.revokeOperator(operator.address);
    const op = await registry.getOperator(operator.address);
    expect(op.active).to.equal(false);
    expect(await registry.isActiveOperator(operator.address)).to.equal(false);
  });

  // SC-005: beaconInterval bounds
  it("SC-005: beaconInterval below 5 or above 3600 reverts", async function () {
    await expect(
      registry.connect(operator).updateConfig(endpoint1, 4)
    ).to.be.revertedWithCustomError(registry, "InvalidBeaconInterval");

    await expect(
      registry.connect(operator).updateConfig(endpoint1, 3601)
    ).to.be.revertedWithCustomError(registry, "InvalidBeaconInterval");
  });

  // SC-006: endpointHash zero reverts
  it("SC-006: zero endpointHash reverts", async function () {
    await expect(
      registry.connect(operator).updateConfig(ethers.ZeroHash, 30)
    ).to.be.revertedWithCustomError(registry, "ZeroEndpointHash");
  });

  // IOT-005: registerDevice emits DeviceRegistered
  it("IOT-005: registerDevice emits DeviceRegistered", async function () {
    const deviceHash = ethers.keccak256(ethers.toUtf8Bytes("device-1"));
    const gatewayHash = ethers.keccak256(ethers.toUtf8Bytes("gateway-1"));
    await expect(registry.connect(operator).registerDevice(deviceHash, gatewayHash))
      .to.emit(registry, "DeviceRegistered")
      .withArgs(deviceHash, gatewayHash, operator.address);
    expect(await registry.registeredDevices(deviceHash)).to.equal(true);
  });
});
