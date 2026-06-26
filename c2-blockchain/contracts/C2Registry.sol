// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

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
    mapping(bytes32 => bool) public registeredDevices;

    error OnlyOwner();
    error OnlyActiveOperator();
    error InvalidBeaconInterval();
    error ZeroEndpointHash();

    event OperatorRegistered(address indexed wallet, bytes32 pubKeyHash);
    event OperatorRevoked(address indexed wallet);
    event ConfigUpdated(
        uint64 indexed version,
        bytes32 endpointHash,
        uint32 beaconIntervalSec,
        address indexed updatedBy
    );
    event DeviceRegistered(bytes32 deviceHash, bytes32 gatewayHash, address registeredBy);

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
    }

    function registerOperator(address wallet, bytes32 pubKeyHash) external onlyOwner {
        operators[wallet] = Operator({wallet: wallet, pubKeyHash: pubKeyHash, active: true});
        operatorCount++;
        emit OperatorRegistered(wallet, pubKeyHash);
    }

    // Owner or any active operator can revoke
    function revokeOperator(address wallet) external {
        if (msg.sender != owner && !operators[msg.sender].active) revert OnlyActiveOperator();
        operators[wallet].active = false;
        emit OperatorRevoked(wallet);
    }

    function updateConfig(bytes32 endpointHash, uint32 beaconIntervalSec) external onlyActiveOperator {
        if (endpointHash == bytes32(0)) revert ZeroEndpointHash();
        if (beaconIntervalSec < 5 || beaconIntervalSec > 3600) revert InvalidBeaconInterval();

        uint64 newVersion = currentConfig.version + 1;
        currentConfig = C2Config({
            endpointHash: endpointHash,
            beaconIntervalSec: beaconIntervalSec,
            version: newVersion
        });
        configHistory[newVersion] = currentConfig;
        emit ConfigUpdated(newVersion, endpointHash, beaconIntervalSec, msg.sender);
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

    function registerDevice(bytes32 deviceHash, bytes32 gatewayHash) external onlyActiveOperator {
        registeredDevices[deviceHash] = true;
        emit DeviceRegistered(deviceHash, gatewayHash, msg.sender);
    }
}
