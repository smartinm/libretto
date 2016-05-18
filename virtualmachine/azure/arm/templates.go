package arm

// Linux is the default arm template to provision a libretto (Linux) vm on Azure
const Linux = `{
  "$schema": "http://schema.management.azure.com/schemas/2014-04-01-preview/deploymentTemplate.json",
  "contentVersion": "1.0.0.0",
  "parameters": {
    "adminUsername": {
      "type": "string"
    },
    "adminPassword": {
      "type": "string"
    },
    "imagePublisher": {
      "type": "string"
    },
    "imageOffer": {
      "type": "string"
    },
    "imageSku": {
      "type": "string"
    },
    "networkSecurityGroup": {
      "type": "string"
    },
    "nicName": {
      "type": "string"
    },    
    "osFileName": {
      "type": "string"
    },
    "publicIPName": {
      "type": "string"
    },
    "sshAuthorizedKey": {
      "type": "string"
    },
    "storageAccountName": {
      "type": "string"
    },
    "storageContainerName": {
      "type": "string"
    },
    "subnetName": {
      "type": "string"
    },
    "virtualNetworkName": {
      "type": "string"
    },
    "vmSize": {
      "type": "string"
    },
    "vmName": {
      "type": "string"
    }
  },
  "variables": {
    "apiVersion": "2015-06-15",
    "location": "[resourceGroup().location]",
    "publicIPAddressType": "Dynamic",
    "subnetRef": "[concat(variables('vnetID'),'/subnets/',parameters('subnetName'))]",
    "sshKeyPath": "[concat('/home/',parameters('adminUsername'),'/.ssh/authorized_keys')]",
    "vnetID": "[resourceId('Microsoft.Network/virtualNetworks', parameters('virtualNetworkName'))]"
  },
  "resources": [
    {
      "apiVersion": "[variables('apiVersion')]",
      "type": "Microsoft.Network/publicIPAddresses",
      "name": "[parameters('publicIPName')]",
      "location": "[variables('location')]",
      "properties": {
        "publicIPAllocationMethod": "[variables('publicIPAddressType')]",
        "dnsSettings": {
          "domainNameLabel": "[parameters('publicIPName')]"
        }
      }
    },
    {
      "apiVersion": "[variables('apiVersion')]",
      "type": "Microsoft.Network/networkInterfaces",
      "name": "[parameters('nicName')]",
      "location": "[variables('location')]",
      "dependsOn": [
        "[concat('Microsoft.Network/publicIPAddresses/', parameters('publicIPName'))]"
      ],
      "properties": {
        "ipConfigurations": [
          {
            "name": "ipconfig",
            "properties": {
              "privateIPAllocationMethod": "Dynamic",
              "publicIPAddress": {
                "id": "[resourceId('Microsoft.Network/publicIPAddresses', parameters('publicIPName'))]"
              },
              "subnet": {
                "id": "[variables('subnetRef')]"
              }
            }
          }
        ],
        "networkSecurityGroup": {
          "id": "[resourceId('Microsoft.Network/networkSecurityGroups', parameters('networkSecurityGroup'))]"
        }
      }
    },
    {
      "apiVersion": "[variables('apiVersion')]",
      "type": "Microsoft.Compute/virtualMachines",
      "name": "[parameters('vmName')]",
      "location": "[variables('location')]",
      "dependsOn": [
        "[concat('Microsoft.Network/networkInterfaces/', parameters('nicName'))]"
      ],
      "properties": {
        "hardwareProfile": {
          "vmSize": "[parameters('vmSize')]"
        },
        "osProfile": {
          "computerName": "[parameters('vmName')]",
          "adminUsername": "[parameters('adminUsername')]",
          "adminPassword": "[parameters('adminPassword')]",
          "linuxConfiguration": {
            "disablePasswordAuthentication": "false"
          }
        },
        "storageProfile": {
          "imageReference": {
            "publisher": "[parameters('imagePublisher')]",
            "offer": "[parameters('imageOffer')]",
            "sku": "[parameters('imageSku')]",
            "version": "latest"
          },
          "osDisk": {
            "name": "osdisk",
            "vhd": {
              "uri": "[concat('http://',parameters('storageAccountName'),'.blob.core.windows.net/',parameters('storageContainerName'),'/', parameters('osFileName'))]"
            },
            "caching": "ReadWrite",
            "createOption": "FromImage"
          }
        },
        "networkProfile": {
          "networkInterfaces": [
            {
              "id": "[resourceId('Microsoft.Network/networkInterfaces', parameters('nicName'))]"
            }
          ]
        },
        "diagnosticsProfile": {
          "bootDiagnostics": {
             "enabled": "false"
          }
        }
      }
    }
  ]
}`
