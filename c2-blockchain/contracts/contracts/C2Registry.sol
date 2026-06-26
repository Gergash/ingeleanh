// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/// @title C2Registry — on-chain config and operator registry for C2 Blockchain-Blindado
contract C2Registry {
    struct Operator {
        address wallet;
        bytes32 pubKeyHash;
        bool active;
    }

    struct C2Config {
        bytes32 endpointHash;
        uint32 beaconIntervalSec;
        uint64 version;
    }

    address public owner;
    mapping(address => Operator) public operators;
    uint256 public operatorCount;
    C2Config public currentConfig;
    mapping(uint64 => C2Config) public configHistory;

    // IoT extension (IOT-005)
    mapping(bytes32 => bool) public registeredDevices;
    mapping(bytes32 => bytes32) public deviceGateway;

    event OperatorRegistered(address indexed wallet, bytes32 pubKeyHash);
    event OperatorRevoked(address indexed wallet);
    event ConfigUpdated(
        uint64 indexed version,
        bytes32 endpointHash,
        uint32 beaconIntervalSec,
        address indexed updatedBy
    );
    event DeviceRegistered(
        bytes32 indexed pubKeyHash,
        bytes32 indexed gatewayHash,
        address indexed registeredBy
    );

    error OnlyOwner();
    error OnlyActiveOperator();
    error ZeroEndpointHash();
    error InvalidBeaconInterval();
    error ZeroWallet();
    error NotAuthorized();

    modifier onlyOwner() {
        if (msg.sender != owner) revert OnlyOwner();
        _;
    }

    modifier onlyActiveOperator() {
        if (!operators[msg.sender].active) revert OnlyActiveOperator();
        _;
    }

    constructor() {
        owner = msg.sender;
        currentConfig = C2Config({endpointHash: bytes32(0), beaconIntervalSec: 0, version: 0});
    }

    function registerOperator(address wallet, bytes32 pubKeyHash) external onlyOwner {
        if (wallet == address(0)) revert ZeroWallet();
        if (!operators[wallet].active) {
            operatorCount++;
        }
        operators[wallet] = Operator(wallet, pubKeyHash, true);
        emit OperatorRegistered(wallet, pubKeyHash);
    }

    function revokeOperator(address wallet) external {
        if (msg.sender != owner) {
            if (!operators[msg.sender].active) revert NotAuthorized();
        }
        operators[wallet].active = false;
        emit OperatorRevoked(wallet);
    }

    function updateConfig(bytes32 endpointHash, uint32 beaconIntervalSec) external onlyActiveOperator {
        if (endpointHash == bytes32(0)) revert ZeroEndpointHash();
        if (beaconIntervalSec < 5 || beaconIntervalSec > 3600) revert InvalidBeaconInterval();

        uint64 newVersion = currentConfig.version + 1;
        C2Config memory config = C2Config({
            endpointHash: endpointHash,
            beaconIntervalSec: beaconIntervalSec,
            version: newVersion
        });
        currentConfig = config;
        configHistory[newVersion] = config;

        emit ConfigUpdated(newVersion, endpointHash, beaconIntervalSec, msg.sender);
    }

    function registerDevice(bytes32 pubKeyHash, bytes32 gatewayHash) external onlyActiveOperator {
        registeredDevices[pubKeyHash] = true;
        deviceGateway[pubKeyHash] = gatewayHash;
        emit DeviceRegistered(pubKeyHash, gatewayHash, msg.sender);
    }

    function revokeDevice(bytes32 pubKeyHash) external onlyActiveOperator {
        registeredDevices[pubKeyHash] = false;
    }

    function getConfig() external view returns (C2Config memory) {
        return currentConfig;
    }

    function getOperator(address wallet) external view returns (Operator memory) {
        return operators[wallet];
    }

    function isActiveOperator(address wallet) external view returns (bool) {
        return operators[wallet].active;
    }
}
